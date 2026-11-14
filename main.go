package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"charm.land/log/v2"
)

func main() {
	log.SetOutput(os.Stdout)

	if err := run(context.Background()); err != nil {
		fmt.Println(err)
		log.Fatal(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {

	// configure Nomad Address
	nomadAddr := os.Getenv("NOMAD_ADDR")

	if nomadAddr == "" {
		// set nomad addr to local agent if it's not set.
		nomadAddr = "http://localhost:4647"
	}
	snapshotOutDir := os.Getenv("SNAPSHOT_DIR")
	if snapshotOutDir == "" {
		snapshotOutDir = "."
	}

	nomadToken := os.Getenv("NOMAD_TOKEN")
	if nomadToken == "" {
		return fmt.Errorf("NOMAD_TOKEN not set.")
	}
	// create Vault client
	// vaultConfig := vault.DefaultConfig()
	// vaultConfig.Address = os.Getenv("VAULT_ADDR")

	// vaultClient, err := vault.NewClient(vaultConfig)
	// if err != nil {
	// 	log.Fatal("Unable to initialize Vault")
	// }

	// vaultClient.SetToken(os.Getenv("VAULT_TOKEN"))

	// create Nomad client by calling NewClient()
	// client, err := nomad.NewClient(&nomad.Config{Address: os.Getenv("NOMAD_ADDR"), Namespace: "default"})
	// if err != nil {
	// 	fmt.Println(err)
	// }

	if err := logTokenInfo(ctx, nomadAddr, nomadToken); err != nil {
		return fmt.Errorf("nomad token preflight: %w", err)
	}

	path, err := saveSnapshot(ctx, nomadAddr, nomadToken, snapshotOutDir)
	if err != nil {
		return err
	}
	log.Info("snapshot saved", "path", path)
	return nil
}

// logTokenInfo verifies that the token passed to the application is accepted by Nomad,
// logs its name and policies, by calling the Nomad API with a context
// and returns an optional error
func logTokenInfo(ctx context.Context, addr, token string) error {
	resp, err := nomadGet(ctx, addr, token, "/v1/acl/token/self")
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status on token self lookup: %s", resp.Status)
	}

	// This will hold the return data of the Nomad token's self lookup
	var tokenSelf struct {
		Name     string
		Type     string
		Policies []string
	}

	// decode the body into the struct
	if err := json.NewDecoder(resp.Body).Decode(&tokenSelf); err != nil {
		return err
	}

	log.Info("Nomad token verified", "name", tokenSelf.Name, "type", tokenSelf.Type)

	return nil

}

// nomadGet is a wrapper around the Nomad API to call Nomad
func nomadGet(ctx context.Context, addr, token, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Nomad-Token", token)
	return http.DefaultClient.Do(req)
}

// saveSnapshot is a function which calls the Nomad API to generate a snapshot
// of the Nomad state
// and write the snapshot to a local cache
// It returns a string representing the path to the snapshot and an optional error
func saveSnapshot(ctx context.Context, addr string, token string, dir string) (string, error) {
	// Call the nomadGet wrapper
	resp, err := nomadGet(ctx, addr, token, "/v1/operator/snapshot")

	if err != nil {
		// if the call fails, we have an empty path and an error
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// if the call returns anything other than an OK status, find out why and log it.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Error("Snapshot request failed", "status", resp.Status, "body", string(body))
		return "", fmt.Errorf("token needs operator:write or operator:snapshot-save permissions")
	}

	// prepare to write the snapshot
	tmp, err := os.CreateTemp(dir, "nomad-snapshot.tmp")
	if err != nil {
		return "", err
	}

	log.Info("Downloading snapshot", "tmp", tmp.Name())

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", fmt.Errorf("Error writing snapshot: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return "", err
	}

	final := filepath.Join(dir, fmt.Sprintf("nomad-snapshot-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z")))

	if err := os.Rename(tmp.Name(), final); err != nil {
		return "", err
	}
	return final, nil
}

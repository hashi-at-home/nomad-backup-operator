package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/jobspec"
	vault "github.com/hashicorp/vault/api"
)

// backupHcl is the name of the backup hcl file
// it will use the go:embed to embed it in the binary

//go:embed backup.hcl
var backupHcl string

// Backup is a struct containing a pointer to a nomad API client
type Backup struct {
	client      *api.Client
	vaultClient *vault.Client
}

// settings is a struct to contain some Nomad job settings
type settings struct {
	JobId       string
	SourceJobId string
	Schedule    string
	TargetDB    string
}

// NewBackup is a function which takes a pointer to a nomad API client
// and returns the address of the Backup
func NewBackup(client *api.Client, vaultClient *vault.Client) *Backup {
	return &Backup{
		client:      client,
		vaultClient: vaultClient,
	}
}

const (
	// BackupFlag is the metadata flag used to trigger auto backup
	BackupFlag = "auto-backup"
	// BackupSchedul is the metadata key used to store the backup schedule
	BackupSchedule = "backup-schedule"
	// BackupTargetDB is the metadata key used to store the backup target database
	BackupTargetDB = "backup-target-db"
)

// OnJob is a function of type pointer to Backup which takes a string and pointer to nomad job
// and returns nil
func (b *Backup) OnJob(eventType string, job *api.Job) {

	// filter out all job IDs that do not have the prefix backup-
	if strings.HasPrefix(*job.ID, "backup-") {
		fmt.Println("Backup job - skipping")
		return
	}

	if strings.Contains(*job.ID, "periodic") {
		fmt.Println("Periodic job - skipping")
		return
	}

	// parse the metdata of the job into settings struct and enabled flag
	settings, enabled := b.parseMeta(*job.ID, job.Meta)

	// remove backup job if the event is a jobDeregister
	if eventType == "jobDeregister" {
		fmt.Println("Job is deregistered: Attempting to remove backup job")
		b.tryRemoveBackupJob(settings.JobId)
		return
	}

	// if auto backup is not enabled, attempt removal of backup job
	if !enabled {
		fmt.Println("Backup is not enabled: Attempting to remove backup job")
		b.tryRemoveBackupJob(settings.JobId)
		return
	}

	// create backup job
	fmt.Println("Registering backup job")
	if err := b.createBackupJob(settings.JobId, settings); err != nil {
		fmt.Printf("Error creating backup job: %v", err)
	}

	// Log end of function
	fmt.Printf("Backup job created: %s", settings.JobId)
}

// parseMeta is a function of type pointer to Backup
// and takes jobID(string), and a map from string to string of the metadata as input
// and returns a settings struct and a bool
func (b *Backup) parseMeta(jobId string, meta map[string]string) (settings, bool) {
	// create a settings struct for the backup job using JobID
	s := settings{
		JobId:       "backup-" + jobId,
		SourceJobId: jobId,
	}

	// Check whether the job has backups enabled in the metadata
	enabled, found := meta[BackupFlag]
	if !found {
		return s, false
	}

	// Check if backup is active
	if active, _ := strconv.ParseBool(enabled); !active {
		return s, false
	}

	// Check whether there is a schedule
	if schedule, found := meta[BackupSchedule]; found {
		s.Schedule = schedule
	}

	// Check whether there is a target database
	if target, found := meta[BackupTargetDB]; found {
		s.TargetDB = target
	}

	return s, true
}

// tryRemoveBackupJob is a function of type pointer to Backup and takes a string as input
// and returns nil
func (b *Backup) tryRemoveBackupJob(jobId string) {
	b.client.Jobs().Deregister(jobId, false, &api.WriteOptions{})
}

// createBackupJob is a function of type pointer to Backup
// and takes a string and settings as input,
// and returns an error
func (b *Backup) createBackupJob(jobId string, s settings) error {
	// create a new job template using [[ ]] as delimiters
	t, err := template.New("").Delims("[[", "]]").Parse(backupHcl)
	if err != nil {
		return err
	}

	// create a buffer to store the rendered hcl
	var buf bytes.Buffer
	// execute template into buffer
	if err := t.Execute(&buf, s); err != nil {
		return err
	}

	// Parse the Nomad job spec from the buffer
	job, err := jobspec.Parse(&buf)
	if err != nil {
		return err
	}

	// Register the job
	_, _, err = b.client.Jobs().Register(job, nil)
	if err != nil {
		return err
	}

	// Log the event
	fmt.Printf("Backup job created: %s", jobId)
	return nil
}

# Nomad backup template job
job "[[ .JobId ]]" {
  datacenters = ["dc1"]
  type = "batch"

  periodic {
    cron = "[[ .Schedule ]]"
    prohibit_overlap = true
  }

  group "backup" {

    count = 1

    task "backup" {
      driver = "docker"

      config = {
        image = "alpine:latest"
        command = "echo"
        args = [
          "backing up [[ .SourceJobId ]]'s [[ .TargetDB ]] database"
        ]
      }
    }
  }
}

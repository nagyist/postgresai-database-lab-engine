/*
2020 © Postgres.ai
*/

// Package logical provides jobs for logical initial operations.
package logical

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	dblabCfg "gitlab.com/postgres-ai/database-lab/pkg/config"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/config"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/engine/postgres/initialize/tools"
	"gitlab.com/postgres-ai/database-lab/pkg/retrieval/options"
)

const (
	// RestoreJobType declares a job type for logical dumping.
	RestoreJobType = "logical-restore"

	// const defines restore options.
	restoreContainerName = "retriever_logical_restore"
	pgDataContainerDir   = "/var/lib/postgresql/pgdata"
	dumpContainerFile    = "/tmp/db.dump"
	defaultParallelJobs  = 1
)

// RestoreJob defines a logical restore job.
type RestoreJob struct {
	name         string
	dockerClient *client.Client
	globalCfg    *dblabCfg.Global
	RestoreOptions
}

// RestoreOptions defines a logical restore options.
type RestoreOptions struct {
	DumpFile     string  `yaml:"dumpLocation"`
	DockerImage  string  `yaml:"dockerImage"`
	DBName       string  `yaml:"dbname"`
	ForceInit    bool    `yaml:"forceInit"`
	ParallelJobs int     `yaml:"parallelJobs"`
	Partial      Partial `yaml:"partial"`
}

// Partial defines tables and rules for a partial logical restore.
type Partial struct {
	Tables []string `yaml:"tables"`
}

// NewJob create a new logical restore job.
func NewJob(cfg config.JobConfig, docker *client.Client, globalCfg *dblabCfg.Global) (*RestoreJob, error) {
	restoreJob := &RestoreJob{
		name:         cfg.Name,
		dockerClient: docker,
		globalCfg:    globalCfg,
	}

	if err := options.Unmarshal(cfg.Options, &restoreJob.RestoreOptions); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal configuration options")
	}

	restoreJob.setDefaults()

	return restoreJob, nil
}

func (r *RestoreJob) setDefaults() {
	// TODO: Default yaml values in tags.
	if r.ParallelJobs == 0 {
		r.ParallelJobs = defaultParallelJobs
	}
}

// Name returns a name of the job.
func (r *RestoreJob) Name() string {
	return r.name
}

// Run starts the job.
func (r *RestoreJob) Run(ctx context.Context) error {
	log.Msg(fmt.Sprintf("Run job: %s. Options: %v", r.Name(), r.RestoreOptions))

	isEmpty, err := tools.IsEmptyDirectory(r.globalCfg.DataDir)
	if err != nil {
		return errors.Wrap(err, "failed to explore the data directory")
	}

	if !isEmpty {
		if !r.ForceInit {
			return errors.New("the data directory is not empty. Use 'forceInit' or empty the data directory")
		}

		log.Msg("The data directory is not empty. Existing data may be overwritten.")
	}

	cont, err := r.dockerClient.ContainerCreate(ctx,
		&container.Config{
			Env: []string{
				"PGDATA=" + pgDataContainerDir,
			},
			Image:       r.RestoreOptions.DockerImage,
			Healthcheck: getContainerHealthConfig(),
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: r.RestoreOptions.DumpFile,
					Target: dumpContainerFile,
				},
				{
					Type:   mount.TypeBind,
					Source: r.globalCfg.DataDir,
					Target: pgDataContainerDir,
				},
			},
		},
		&network.NetworkingConfig{},
		restoreContainerName,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create container")
	}

	defer func() {
		if err := r.dockerClient.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			log.Err("Failed to remove container: ", err)

			return
		}

		log.Msg(fmt.Sprintf("Stop container: %s. ID: %v", restoreContainerName, cont.ID))
	}()

	if err := r.dockerClient.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
		return errors.Wrap(err, "failed to start container")
	}

	log.Msg(fmt.Sprintf("Running container: %s. ID: %v", restoreContainerName, cont.ID))

	if err := tools.CheckContainerReadiness(ctx, r.dockerClient, cont.ID); err != nil {
		return errors.Wrap(err, "failed to readiness check")
	}

	log.Msg("Running restore command")

	execCommand, err := r.dockerClient.ContainerExecCreate(ctx, cont.ID, types.ExecConfig{
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          r.buildLogicalRestoreCommand(),
	})

	if err != nil {
		return errors.Wrap(err, "failed to create an exec command")
	}

	if len(r.Partial.Tables) > 0 {
		log.Msg("Partial restore will be run. Tables for restoring: ", strings.Join(r.Partial.Tables, ", "))
	}

	if err := r.dockerClient.ContainerExecStart(ctx, execCommand.ID, types.ExecStartCheck{Tty: true}); err != nil {
		return errors.Wrap(err, "failed to run the exec command")
	}

	if err := tools.InspectCommandResponse(ctx, r.dockerClient, cont.ID, execCommand.ID); err != nil {
		return errors.Wrap(err, "failed to exec the restore command")
	}

	log.Msg("Restoring job has been finished")

	return nil
}

func (r *RestoreJob) buildLogicalRestoreCommand() []string {
	restoreCmd := []string{"pg_restore", "-U", defaultUsername, "-C"}

	if r.ForceInit {
		restoreCmd = append(restoreCmd, "-d", defaultDBName, "--clean", "--if-exists")
	} else {
		restoreCmd = append(restoreCmd, "-d", r.RestoreOptions.DBName)
	}

	restoreCmd = append(restoreCmd, "-j", strconv.Itoa(r.ParallelJobs))

	for _, table := range r.Partial.Tables {
		restoreCmd = append(restoreCmd, "-t", table)
	}

	restoreCmd = append(restoreCmd, dumpContainerFile)

	return restoreCmd
}
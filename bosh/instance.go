package bosh

import (
	"fmt"
	"io"
	"strings"
	"errors"

	"bytes"
	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/hashicorp/go-multierror"
	"github.com/pivotal-cf/pcf-backup-and-restore/backuper"
)

type DeployedInstance struct {
	director.Deployment
	InstanceGroupName             string
	BackupAndRestoreInstanceIndex string
	BoshInstanceID                string
	SSHConnection
	Logger
	backupable                    *bool
	restorable                    *bool
	unlockable                    *bool
	lockable                      *bool
	BackupAndRestoreScripts
}

//go:generate counterfeiter -o fakes/fake_ssh_connection.go . SSHConnection
type SSHConnection interface {
	Stream(cmd string, writer io.Writer) ([]byte, int, error)
	StreamStdin(cmd string, reader io.Reader) ([]byte, []byte, int, error)
	Run(cmd string) ([]byte, []byte, int, error)
	Cleanup() error
	Username() string
}

func NewBoshInstance(instanceGroupName, instanceIndex, instanceID string, connection SSHConnection, deployment director.Deployment, logger Logger, scripts BackupAndRestoreScripts) backuper.Instance {
	return &DeployedInstance{
		BackupAndRestoreInstanceIndex:     instanceIndex,
		InstanceGroupName: instanceGroupName,
		BoshInstanceID:        instanceID,
		SSHConnection:     connection,
		Deployment:        deployment,
		Logger:            logger,
		BackupAndRestoreScripts: scripts,
	}
}

func (d *DeployedInstance) IsBackupable() (bool, error) {
	if d.backupable != nil {
		return *d.backupable, nil
	}
	_, _, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-backup", "check for backup scripts")
	if err != nil {
		return false, err
	}
	backupable := exitCode == 0
	d.backupable = &backupable

	return *d.backupable, err
}

func (d *DeployedInstance) IsPostBackupUnlockable() (bool, error) {
	if d.unlockable != nil {
		return *d.unlockable, nil
	}
	_, _, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-post-backup-unlock", "check for post-backup-unlock scripts")
	if err != nil {
		return false, err
	}
	unlockable := exitCode == 0
	d.unlockable = &unlockable

	return *d.unlockable, err
}

func (d *DeployedInstance) IsPreBackupLockable() (bool, error) {
	if d.lockable != nil {
		return *d.lockable, nil
	}
	_, _, exitCode, err := d.logAndRun("sudo ls /var/vcap/jobs/*/bin/p-pre-backup-lock", "check for pre-backup-lock scripts")
	if err != nil {
		return false, err
	}

	lockable := exitCode == 0
	d.lockable = &lockable

	return *d.lockable, err
}

func (d *DeployedInstance) PreBackupLock() error {
	d.Logger.Info("", "Locking %s/%s for backup...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, script := range d.BackupAndRestoreScripts.PreBackupLockOnly(){
		d.Logger.Debug("", "> %s", script)

		jobName, _ := script.JobName()

		stdout, stderr, exitCode, err := d.logAndRun(fmt.Sprintf("sudo %s", script), "backup")

		if err != nil {
			d.Logger.Error("", fmt.Sprintf(
				"Error attempting to run pre backup lock script for job %s on %s/%s. Error: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				err.Error(),
			))
			foundErrors = multierror.Append(foundErrors, err)
		}

		if exitCode != 0 {
			errorString := fmt.Sprintf(
				"Pre backup lock script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				stdout,
				stderr,
			)

			foundErrors = multierror.Append(foundErrors, errors.New(errorString))

			d.Logger.Error("", errorString)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) Backup() error {
	d.Logger.Info("", "Backing up %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, job := range d.jobs().Backupable(){
		d.Logger.Debug("", "> %s", job.BackupScript())

		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo mkdir -p %s && sudo ARTIFACT_DIRECTORY=%s/ %s",
				job.ArtifactDirectory(),
				job.ArtifactDirectory(),
				job.BackupScript(),
			),
			"backup",
		)

		if err != nil {
			d.Logger.Error("", fmt.Sprintf(
				"Error attempting to run backup script for job %s on %s/%s. Error: %s",
				job.Name(),
				d.InstanceGroupName,
				d.BoshInstanceID,
				err.Error(),
			))
			foundErrors = multierror.Append(foundErrors, err)
		}

		if exitCode != 0 {
			errorString := fmt.Sprintf(
				"Backup script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
				job.Name(),
				d.InstanceGroupName,
				d.BoshInstanceID,
				stdout,
				stderr,
			)

			foundErrors = multierror.Append(foundErrors, errors.New(errorString))

			d.Logger.Error("", errorString)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) PostBackupUnlock() error {
	d.Logger.Info("", "Unlocking %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var foundErrors error

	for _, script := range d.BackupAndRestoreScripts.PostBackupUnlockOnly(){
		d.Logger.Debug("", "> %s", script)

		jobName, _ := script.JobName()

		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo %s",
				script,
			),
			"unlock",
		)

		if err != nil {
			d.Logger.Error("", fmt.Sprintf(
				"Error attempting to run unlock script for job %s on %s/%s. Error: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				err.Error(),
			))
			foundErrors = multierror.Append(foundErrors, err)
		}

		if exitCode != 0 {
			errorString := fmt.Sprintf(
				"Unlock script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				stdout,
				stderr,
			)

			foundErrors = multierror.Append(foundErrors, errors.New(errorString))

			d.Logger.Error("", errorString)
		}
	}

	if foundErrors != nil {
		return foundErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) Restore() error {
	d.Logger.Info("", "Restoring to %s/%s...", d.InstanceGroupName, d.BoshInstanceID)

	var restoreErrors error

	for _, script := range d.BackupAndRestoreScripts.RestoreOnly() {
		d.Logger.Debug("", "> %s", script)

		jobName, _ := script.JobName()
		artifactDirectory := fmt.Sprintf("/var/vcap/store/backup/%s", jobName)
		stdout, stderr, exitCode, err := d.logAndRun(
			fmt.Sprintf(
				"sudo ARTIFACT_DIRECTORY=%s/ %s",
				artifactDirectory,
				script,
			),
			"restore",
		)

		if err != nil {
			d.Logger.Error("", fmt.Sprintf(
				"Error attempting to run restore script for job %s on %s/%s. Error: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				err.Error(),
			))
			restoreErrors = multierror.Append(restoreErrors, err)
		}

		if exitCode != 0 {
			errorString := fmt.Sprintf(
				"Restore script for job %s failed on %s/%s.\nStdout: %s\nStderr: %s",
				jobName,
				d.InstanceGroupName,
				d.BoshInstanceID,
				stdout,
				stderr,
			)

			restoreErrors = multierror.Append(restoreErrors, errors.New(errorString))

			d.Logger.Error("", errorString)
		}
	}

	if restoreErrors != nil {
		return restoreErrors
	}

	d.Logger.Info("", "Done.")
	return nil
}

func (d *DeployedInstance) StreamBackupFromRemote(writer io.Writer) error {
	d.Logger.Debug("", "Streaming backup from instance %s/%s", d.InstanceGroupName, d.BoshInstanceID)
	stderr, exitCode, err := d.Stream("sudo tar -C /var/vcap/store/backup -zc .", writer)

	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running instance backup scripts. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Instance backup scripts returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d *DeployedInstance) StreamBackupToRemote(reader io.Reader) error {
	stdout, stderr, exitCode, err := d.logAndRun("sudo mkdir -p /var/vcap/store/backup/", "create backup directory on remote")

	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("Creating backup directory on the remote returned %d. Error: %s", exitCode, stderr)
	}

	d.Logger.Debug("", "Streaming backup to instance %s/%s", d.InstanceGroupName, d.BoshInstanceID)
	stdout, stderr, exitCode, err = d.StreamStdin("sudo sh -c 'tar -C /var/vcap/store/backup -zx'", reader)

	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error streaming backup to remote instance. Exit code %d, error %s", exitCode, err.Error())
	}

	if exitCode != 0 {
		return fmt.Errorf("Streaming backup to remote returned %d. Error: %s", exitCode, stderr)
	}

	return err
}

func (d *DeployedInstance) BackupChecksum() (backuper.BackupChecksum, error) {
	stdout, stderr, exitCode, err := d.logAndRun("cd /var/vcap/store/backup; sudo sh -c 'find . -type f | xargs shasum'", "checksum")

	if err != nil {
		return nil, err
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("Instance checksum returned %d. Error: %s", exitCode, stderr)
	}

	return convertShasToMap(string(stdout)), nil
}

func (d *DeployedInstance) IsRestorable() (bool, error) {
	if d.restorable != nil {
		return *d.restorable, nil
	}
	_, _, exitCode, err := d.logAndRun("ls /var/vcap/jobs/*/bin/p-restore", "check for restore scripts")
	if err != nil {
		return false, err
	}

	restorable := exitCode == 0
	d.restorable = &restorable

	return *d.restorable, err
}

func (d *DeployedInstance) BackupSize() (string, error) {
	stdout, stderr, exitCode, err := d.logAndRun("sudo du -sh /var/vcap/store/backup/ | cut -f1", "check backup size")

	if exitCode != 0 {
		return "", fmt.Errorf("Unable to check size of backup: %s", stderr)
	}

	size := strings.TrimSpace(string(stdout))
	return size, err
}

func (d *DeployedInstance) Cleanup() error {
	var errs error
	d.Logger.Debug("", "Cleaning up SSH connection on instance %s %s", d.InstanceGroupName, d.BoshInstanceID)
	removeArtifactError := d.removeBackupArtifacts()
	if removeArtifactError != nil {
		errs = multierror.Append(errs, removeArtifactError)
	}
	cleanupSSHError := d.CleanUpSSH(director.NewAllOrInstanceGroupOrInstanceSlug(d.InstanceGroupName, d.BoshInstanceID), director.SSHOpts{Username: d.SSHConnection.Username()})
	if cleanupSSHError != nil {
		errs = multierror.Append(errs, cleanupSSHError)
	}
	return errs
}

func (d *DeployedInstance) logAndRun(cmd, label string) ([]byte, []byte, int, error) {
	d.Logger.Debug("", "Running %s on %s/%s", label, d.InstanceGroupName, d.BoshInstanceID)

	stdout, stderr, exitCode, err := d.Run(cmd)
	d.Logger.Debug("", "Stdout: %s", string(stdout))
	d.Logger.Debug("", "Stderr: %s", string(stderr))

	if err != nil {
		d.Logger.Debug("", "Error running %s on instance %s/%s. Exit code %d, error: %s", label, d.InstanceGroupName, d.BoshInstanceID, exitCode, err.Error())
	}

	return stdout, stderr, exitCode, err
}

func (d *DeployedInstance) filesPresent(path string) {
	d.Logger.Debug("", "Listing contents of %s on %s/%s", path, d.InstanceGroupName, d.BoshInstanceID)

	stdout, _, _, _ := d.Run("sudo ls " + path)
	stdout = bytes.TrimSpace(stdout)
	files := strings.Split(string(stdout), "\n")

	for _, f := range files {
		d.Logger.Debug("", "> %s", f)
	}
}

func (d *DeployedInstance) Name() string {
	return d.InstanceGroupName
}

func (d *DeployedInstance) Index() string {
	return d.BackupAndRestoreInstanceIndex
}

func (d *DeployedInstance) ID() string {
	return d.BoshInstanceID
}

func (d *DeployedInstance) removeBackupArtifacts() error {
	_, _, _, err := d.logAndRun("sudo rm -rf /var/vcap/store/backup", "remove backup artifacts")
	return err
}

func (d *DeployedInstance) jobs() Jobs {
	return NewJobs(d.BackupAndRestoreScripts)
}

func convertShasToMap(shas string) map[string]string {
	mapOfSha := map[string]string{}
	shas = strings.TrimSpace(shas)
	if shas == "" {
		return mapOfSha
	}
	for _, line := range strings.Split(shas, "\n") {
		parts := strings.SplitN(line, " ", 2)
		filename := strings.TrimSpace(parts[1])
		if filename == "-" {
			continue
		}
		mapOfSha[filename] = parts[0]
	}
	return mapOfSha
}

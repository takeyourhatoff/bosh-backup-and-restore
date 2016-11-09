package backuper

import (
	"fmt"
	"io"
)

func New(bosh BoshDirector, artifactCreator ArtifactCreator, logger Logger) *Backuper {
	return &Backuper{
		BoshDirector:    bosh,
		ArtifactCreator: artifactCreator,
		Logger:          logger,
	}
}

//go:generate counterfeiter -o fakes/fake_artifact_creator.go . ArtifactCreator
type ArtifactCreator func(string) (Artifact, error)

//go:generate counterfeiter -o fakes/fake_artifact.go . Artifact
type Artifact interface {
	CreateFile(Instance) (io.WriteCloser, error)
	AddChecksum(Instance, string) error
	CalculateChecksum(Instance) (string, error)
	DeploymentMatches(string, []Instance) (bool, error)
}

//go:generate counterfeiter -o fakes/fake_logger.go . Logger
type Logger interface {
	Debug(tag, msg string, args ...interface{})
	Info(tag, msg string, args ...interface{})
	Warn(tag, msg string, args ...interface{})
	Error(tag, msg string, args ...interface{})
}

type Backuper struct {
	BoshDirector
	ArtifactCreator
	Logger
}

//Backup checks if a deployment has backupable instances and backs them up.
func (b Backuper) Backup(deploymentName string) error {
	b.Logger.Info("", "Starting backup of %s...\n", deploymentName)

	b.Logger.Info("", "Finding instances with backup scripts...")
	instances, err := b.FindInstances(deploymentName)
	if err != nil {
		return err
	}
	b.Logger.Info("", "Done.")
	defer instances.Cleanup()

	backupableInstances, err := instances.AllBackupable()
	if err != nil {
		return err
	}
	if backupableInstances.IsEmpty() {
		return fmt.Errorf("Deployment '%s' has no backup scripts", deploymentName)
	}

	artifact, err := b.ArtifactCreator(deploymentName)
	if err != nil {
		return err
	}

	if err = backupableInstances.Backup(); err != nil {
		return err
	}

	//TODO: Refactor me, maybe
	for _, instance := range backupableInstances {
		writer, err := artifact.CreateFile(instance)

		if err != nil {
			return err
		}

		size, err := instance.BackupSize()
		if err != nil {
			return err
		}

		b.Logger.Info("", "Copying backup -- %s uncompressed -- from %s-%s...", size, instance.Name(), instance.ID())
		if err := instance.StreamBackupTo(writer); err != nil {
			return err
		}

		if err := writer.Close(); err != nil {
			return err
		}

		localChecksum, err := artifact.CalculateChecksum(instance)
		if err != nil {
			return err
		}

		remoteChecksum, err := instance.BackupChecksum()
		if err != nil {
			return err
		}
		if localChecksum != remoteChecksum {
			return fmt.Errorf("Backup artifact is corrupted, checksum failed for %s:%s, remote checksum: %s, local checksum: %s", instance.Name(), instance.ID(), remoteChecksum, localChecksum)
		}

		artifact.AddChecksum(instance, localChecksum)
		b.Logger.Info("", "Done.")
	}

	b.Logger.Info("", "Completed backup of %s\n", deploymentName)
	return nil
}

func (b Backuper) Restore(deploymentName string) error {
	instances, err := b.FindInstances(deploymentName)
	if err != nil {
		return err
	}

	defer instances.Cleanup()

	var restorableInstances []Instance

	for _, inst := range instances {
		restorable, err := inst.IsRestorable()
		if err != nil {
			return fmt.Errorf("Error occurred while checking if deployment could be restored: %s", err)
		}

		if restorable {
			restorableInstances = append(restorableInstances, inst)
		}
	}

	if len(restorableInstances) == 0 {
		return fmt.Errorf("Deployment '%s' has no restore scripts", deploymentName)
	}

	artifact, _ := b.ArtifactCreator(deploymentName)
	match, err := artifact.DeploymentMatches(deploymentName, instances)

	if err != nil {
		return fmt.Errorf("Unable to check if deployment '%s' matches the structure of the provided backup", deploymentName)
	}

	if match != true {
		return fmt.Errorf("Deployment '%s' does not match the structure of the provided backup", deploymentName)
	}

	for _, instance := range restorableInstances {
		instance.Restore()
	}

	return nil
}

//go:generate counterfeiter -o fakes/fake_bosh_director.go . BoshDirector
type BoshDirector interface {
	FindInstances(deploymentName string) (Instances, error)
}

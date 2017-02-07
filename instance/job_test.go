package instance_test

import (
	"github.com/pivotal-cf/pcf-backup-and-restore/instance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Job", func() {
	var job instance.Job
	var jobScripts instance.BackupAndRestoreScripts
	var metadata instance.Metadata

	BeforeEach(func() {
		jobScripts = instance.BackupAndRestoreScripts{
			"/var/vcap/jobs/foo/bin/p-restore",
			"/var/vcap/jobs/foo/bin/p-backup",
			"/var/vcap/jobs/foo/bin/p-pre-backup-lock",
			"/var/vcap/jobs/foo/bin/p-post-backup-unlock",
		}
		metadata = instance.Metadata{}
	})

	JustBeforeEach(func() {
		job = instance.NewJob(jobScripts, metadata)
	})

	Describe("ArtifactDirectory", func() {
		It("calculates the blob directory based on the name", func() {
			Expect(job.ArtifactDirectory()).To(Equal("/var/vcap/store/backup/foo"))
		})

		Context("when an blob name is provided", func() {
			var jobWithName instance.Job

			JustBeforeEach(func() {
				jobWithName = instance.NewJob(jobScripts, instance.Metadata{
					BackupName: "a-bosh-backup",
				})
			})

			It("calculates the blob directory based on the blob name", func() {
				Expect(jobWithName.ArtifactDirectory()).To(Equal("/var/vcap/store/backup/a-bosh-backup"))
			})
		})
	})

	Describe("BackupScript", func() {
		It("returns the backup script", func() {
			Expect(job.BackupScript()).To(Equal(instance.Script("/var/vcap/jobs/foo/bin/p-backup")))
		})
		Context("no backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns nil", func() {
				Expect(job.BackupScript()).To(BeEmpty())
			})
		})
	})

	Describe("BackupBlobName", func() {
		Context("the job has a custom backup blob name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "fool",
				}
			})

			It("returns the job's custom backup blob name", func() {
				Expect(job.BackupBlobName()).To(Equal("fool"))
			})
		})

		Context("the job does not have a custom backup blob name", func() {
			It("returns empty string", func() {
				Expect(job.BackupBlobName()).To(Equal(""))
			})
		})
	})

	Describe("RestoreBlobName", func() {
		Context("the job has a custom backup blob name", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "bard",
				}
			})

			It("returns the job's custom backup blob name", func() {
				Expect(job.RestoreBlobName()).To(Equal("bard"))
			})
		})

		Context("the job does not have a custom backup blob name", func() {
			It("returns empty string", func() {
				Expect(job.RestoreBlobName()).To(Equal(""))
			})
		})
	})

	Describe("HasBackup", func() {
		It("returns true", func() {
			Expect(job.HasBackup()).To(BeTrue())
		})

		Context("no backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns false", func() {
				Expect(job.HasBackup()).To(BeFalse())
			})
		})
	})

	Describe("PreBackupScript", func() {
		It("returns the pre-backup script", func() {
			Expect(job.PreBackupScript()).To(Equal(instance.Script("/var/vcap/jobs/foo/bin/p-pre-backup-lock")))
		})
		Context("no pre-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns nil", func() {
				Expect(job.PreBackupScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasPreBackup", func() {
		It("returns true", func() {
			Expect(job.HasPreBackup()).To(BeTrue())
		})

		Context("no pre-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns false", func() {
				Expect(job.HasPreBackup()).To(BeFalse())
			})
		})
	})

	Describe("PostBackupScript", func() {
		It("returns the post-backup script", func() {
			Expect(job.PostBackupScript()).To(Equal(instance.Script("/var/vcap/jobs/foo/bin/p-post-backup-unlock")))
		})
		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns nil", func() {
				Expect(job.PostBackupScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasPostBackup", func() {
		It("returns true", func() {
			Expect(job.HasPostBackup()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-restore"}
			})
			It("returns false", func() {
				Expect(job.HasPostBackup()).To(BeFalse())
			})
		})
	})

	Describe("RestoreScript", func() {
		It("returns the post-backup script", func() {
			Expect(job.RestoreScript()).To(Equal(instance.Script("/var/vcap/jobs/foo/bin/p-restore")))
		})
		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}
			})
			It("returns nil", func() {
				Expect(job.RestoreScript()).To(BeEmpty())
			})
		})
	})

	Describe("HasRestore", func() {
		It("returns true", func() {
			Expect(job.HasRestore()).To(BeTrue())
		})

		Context("no post-backup scripts exist", func() {
			BeforeEach(func() {
				jobScripts = instance.BackupAndRestoreScripts{"/var/vcap/jobs/foo/bin/p-backup"}
			})
			It("returns false", func() {
				Expect(job.HasRestore()).To(BeFalse())
			})
		})
	})

	Describe("HasNamedBackupBlob", func() {
		It("returns false", func() {
			Expect(job.HasNamedBackupBlob()).To(BeFalse())
		})

		Context("when the job has a named backup blob", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "foo",
				}
			})

			It("returns true", func() {
				Expect(job.HasNamedBackupBlob()).To(BeTrue())
			})
		})

		Context("when the job has a named restore blob", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "foo",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedBackupBlob()).To(BeFalse())
			})
		})
	})

	Describe("HasNamedRestoreBlob", func() {
		It("returns false", func() {
			Expect(job.HasNamedRestoreBlob()).To(BeFalse())
		})

		Context("when the job has a named restore blob", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					RestoreName: "foo",
				}
			})

			It("returns true", func() {
				Expect(job.HasNamedRestoreBlob()).To(BeTrue())
			})
		})

		Context("when the job has a named backup blob", func() {
			BeforeEach(func() {
				metadata = instance.Metadata{
					BackupName: "foo",
				}
			})

			It("returns false", func() {
				Expect(job.HasNamedRestoreBlob()).To(BeFalse())
			})
		})
	})
})

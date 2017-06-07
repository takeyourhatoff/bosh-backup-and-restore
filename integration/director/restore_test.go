package director

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pivotal-cf/bosh-backup-and-restore/testcluster"

	"github.com/onsi/gomega/gexec"

	"os/exec"

	"path"

	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Restore", func() {
	var restoreWorkspace string
	var session *gexec.Session
	var directorAddress, directorIP string
	var artifactName string

	BeforeEach(func() {
		var err error
		restoreWorkspace, err = ioutil.TempDir(".", "restore-workspace-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(restoreWorkspace)).To(Succeed())
	})

	JustBeforeEach(func() {
		session = binary.Run(
			restoreWorkspace,
			[]string{"BOSH_CLIENT_SECRET=admin"},
			"director",
			"--host", directorAddress,
			"--username", "foobar",
			"--private-key-path", pathToPrivateKeyFile,
			"--debug",
			"restore",
		)
	})

	Context("When there is a director instance", func() {
		var directorInstance *testcluster.Instance

		BeforeEach(func() {
			directorInstance = testcluster.NewInstance()
			directorInstance.CreateUser("foobar", readFile(pathToPublicKeyFile))
			directorAddress = directorInstance.Address()
			directorIP = directorInstance.IP()
			artifactName = "director-backup-integration"
		})

		AfterEach(func() {
			directorInstance.DieInBackground()
		})

		Context("and there is a restore script", func() {
			BeforeEach(func() {
				command := exec.Command("cp", "-r", "../../fixtures/director-backup-integration", path.Join(restoreWorkspace, directorAddress))
				cpFiles, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(cpFiles).Should(gexec.Exit())

				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/restore")
			})

			Context("and the restore script succeeds", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", `#!/usr/bin/env sh
set -u

mkdir -p /var/vcap/store/bosh/
cat $BBR_ARTIFACT_DIRECTORY/backup > /var/vcap/store/bosh/restored_file
`)
				})

				It("successfully restores to the director", func() {
					By("exiting zero", func() {
						Expect(session.ExitCode()).To(BeZero())
					})

					By("running the restore script successfully", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bosh/restored_file")).To(BeTrue())
						Expect(directorInstance.GetFileContents("/var/vcap/store/bosh/restored_file")).To(ContainSubstring(`this is a backup`))
					})

					By("cleaning up backup artifacts from the remote", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeFalse())
					})
				})
			})

			Context("but the restore script fails", func() {
				BeforeEach(func() {
					directorInstance.CreateScript("/var/vcap/jobs/bosh/bin/bbr/restore", "echo 'NOPE!'; exit 1")
				})

				It("fails to backup the director", func() {
					By("returning exit code 1", func() {
						Expect(session.ExitCode()).To(Equal(1))
					})
				})
			})

			Context("but the artifact directory already exists", func() {
				BeforeEach(func() {
					directorInstance.CreateDir("/var/vcap/store/bbr-backup")
				})

				It("fails to backup the director", func() {
					By("exiting non-zero", func() {
						Expect(session.ExitCode()).NotTo(BeZero())
					})

					By("printing a log message saying the director instance cannot be backed up", func() {
						Expect(string(session.Err.Contents())).To(ContainSubstring("Directory /var/vcap/store/bbr-backup already exists on instance bosh/0"))
					})

					By("not deleting the existing artifact directory", func() {
						Expect(directorInstance.FileExists("/var/vcap/store/bbr-backup")).To(BeTrue())
					})
				})
			})
		})

		Context("but there are no restore scripts", func() {
			BeforeEach(func() {
				command := exec.Command("cp", "-r", "../../fixtures/director-backup-integration", path.Join(restoreWorkspace, directorAddress))
				cpFiles, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(cpFiles).Should(gexec.Exit())

				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/backup")
				directorInstance.CreateFiles("/var/vcap/jobs/bosh/bin/bbr/not-a-restore-script")
			})

			It("fails to backup the director", func() {
				By("returning exit code 1", func() {
					Expect(session.ExitCode()).To(Equal(1))
				})

				By("printing an error", func() {
					Expect(string(session.Err.Contents())).To(ContainSubstring(fmt.Sprintf("Deployment '%s' has no restore scripts", directorIP)))
				})

				By("saving the stack trace into a file", func() {
					files, err := filepath.Glob(filepath.Join(restoreWorkspace, "bbr-*.err.log"))
					Expect(err).NotTo(HaveOccurred())
					logFilePath := files[0]
					_, err = os.Stat(logFilePath)
					Expect(os.IsNotExist(err)).To(BeFalse())
					stackTrace, err := ioutil.ReadFile(logFilePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(gbytes.BufferWithBytes(stackTrace)).To(gbytes.Say("main.go"))
				})
			})
		})
	})

	Context("When the director does not resolve", func() {
		BeforeEach(func() {
			command := exec.Command("cp", "-r", "../../fixtures/director-backup-integration", path.Join(restoreWorkspace, "does-not-resolve"))
			cpFiles, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(cpFiles).Should(gexec.Exit())

			directorAddress = "does-not-resolve"
		})

		It("fails to restore the director", func() {
			By("returning exit code 1", func() {
				Expect(session.ExitCode()).To(Equal(1))
			})

			By("printing an error", func() {
				Expect(string(session.Err.Contents())).To(ContainSubstring("no such host"))
			})
		})
	})
})

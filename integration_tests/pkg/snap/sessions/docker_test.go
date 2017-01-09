package sessions

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/intelsdi-x/snap/scheduler/wmap"
	"github.com/intelsdi-x/swan/integration_tests/test_helpers"
	"github.com/intelsdi-x/swan/pkg/executor"
	"github.com/intelsdi-x/swan/pkg/kubernetes"
	"github.com/intelsdi-x/swan/pkg/snap"
	"github.com/intelsdi-x/swan/pkg/snap/sessions/docker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSnapDockerSession(t *testing.T) {
	Convey("Preparing Snap and Kubernetes enviroment", t, func() {
		snapteld := testhelpers.NewSnapteld()
		err := snapteld.Start()
		So(err, ShouldBeNil)

		defer snapteld.CleanAndEraseOutput()
		defer snapteld.Stop()

		snapteldAddress := fmt.Sprintf("http://%s:%d", "127.0.0.1", snapteld.Port())

		// Load plugins.
		loaderConfig := snap.DefaultPluginLoaderConfig()
		loaderConfig.SnapteldAddress = snapteldAddress
		loader, err := snap.NewPluginLoader(loaderConfig)
		So(err, ShouldBeNil)

		err = loader.Load(snap.DockerCollector, snap.FilePublisher)
		So(err, ShouldBeNil)
		publisherPluginName, _, err := snap.GetPluginNameAndType(snap.FilePublisher)
		So(err, ShouldBeNil)

		resultsFile, err := ioutil.TempFile("", "session_test")
		So(err, ShouldBeNil)
		resultsFileName := resultsFile.Name()
		defer os.Remove(resultsFileName)
		resultsFile.Close()

		publisher := wmap.NewPublishNode(publisherPluginName, snap.PluginAnyVersion)
		publisher.AddConfigItem("file", resultsFileName)

		// Run Kubernetes
		exec := executor.NewLocal()
		config, err := kubernetes.UniqueConfig()
		So(err, ShouldBeNil)
		kubernetesLauncher := kubernetes.New(exec, exec, config)
		kubernetesHandle, err := kubernetesLauncher.Launch()
		So(err, ShouldBeNil)
		So(kubernetesHandle, ShouldNotBeNil)
		defer kubernetesHandle.EraseOutput()
		defer kubernetesHandle.Clean()
		defer kubernetesHandle.Stop()

		// Waiting for Kubernetes Executor.
		kubernetesConfig := executor.DefaultKubernetesConfig()
		kubernetesConfig.Address = fmt.Sprintf("127.0.0.1:%d", config.KubeAPIPort)
		kubeExecutor, err := executor.NewKubernetes(kubernetesConfig)
		So(err, ShouldBeNil)

		podHandle, err := kubeExecutor.Execute("stress -c 1 -t 600")
		So(err, ShouldBeNil)
		defer podHandle.EraseOutput()
		defer podHandle.Clean()
		defer podHandle.Stop()

		// Prepare Docker Session.
		/*experimentID, err := uuid.NewV4()
		So(err, ShouldBeNil)
		phaseID, err := uuid.NewV4()
		So(err, ShouldBeNil)*/

		Convey("Launching Docker Session", func() {
			dockerConfig := docker.DefaultConfig()
			dockerConfig.SnapteldAddress = snapteldAddress
			dockerConfig.Publisher = publisher
			dockerLauncher, err := docker.NewSessionLauncher(dockerConfig)
			So(err, ShouldBeNil)
			dockerHandle, err := dockerLauncher.LaunchSession(
				nil,
				/*phase.Session{
					ExperimentID: experimentID.String(),
					PhaseID:      phaseID.String(),
					RepetitionID: 1,
				},*/
				"foo:bar",
			)
			So(err, ShouldBeNil)
			So(dockerHandle.IsRunning(), ShouldBeTrue)
			dockerHandle.Wait()
			time.Sleep(5 * time.Second) // One hit does not always yield results.
			dockerHandle.Stop()

			// one measurement should contains more then one metric.
			oneMeasurement, err := testhelpers.GetOneMeasurementFromFile(resultsFileName)
			So(err, ShouldBeNil)
			So(len(oneMeasurement), ShouldBeGreaterThan, 0)

			Convey("There should be CPU results of docker containers on Kubernetes", func() {
				requestedMetric, err := testhelpers.GetMetric(`/intel/docker/root/stats/cgroups/cpu_stats/cpu_usage/total_usage`, oneMeasurement)
				So(err, ShouldBeNil)
				So(requestedMetric.Data.(float64), ShouldBeGreaterThan, 0)
				So(requestedMetric.Tags[`foo`], ShouldEqual, `bar`)
			})

			Convey("There should be Memory results of docker containers on Kubernetes", func() {
				requestedMetric, err := testhelpers.GetMetric(`/intel/docker/root/stats/cgroups/memory_stats/usage/usage`, oneMeasurement)
				So(err, ShouldBeNil)
				So(requestedMetric.Data.(float64), ShouldBeGreaterThan, 0)
				So(requestedMetric.Tags[`foo`], ShouldEqual, `bar`)
			})
		})
	})
}

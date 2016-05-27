package isolation

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCPUSelect(t *testing.T) {

	// cpuDiscovered collect CPU topology.
	var cpus CPUInfo

	cpus.Discover()

	threadset := NewIntSet()

	Convey("Should provide CPUSelect() to not return nil when requesting zero cpu", t, func() {

		So(threadset.CPUSelect(0, ShareLLCButNotL1L2), ShouldNotBeNil)

		Convey("Should have length zero", func() {
			So(threadset, ShouldHaveLength, 0)
		})
	})

	Convey("Should provide CPUSelect() to return nil and correct cpu ids", t, func() {

		So(threadset.CPUSelect(cpus.PhysicalCores, ShareLLCButNotL1L2), ShouldBeNil)

		Convey("Should have length", func() {
			So(threadset, ShouldHaveLength, cpus.PhysicalCores)
		})

		Convey("It should contain all core ids", func() {
			for i := 0; i < cpus.PhysicalCores; i++ {
				So(threadset.Contains(i), ShouldBeTrue)
			}
		})
	})

	Convey("Should provide CPUSelect() to not return nil when requesting more cores than a socket has", t, func() {

		So(threadset.CPUSelect(cpus.PhysicalCores+1, ShareLLCButNotL1L2), ShouldNotBeNil)

	})

}

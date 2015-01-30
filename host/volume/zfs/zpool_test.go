package zfs

import (
	"fmt"
	"math"
	"os"

	. "github.com/flynn/flynn/Godeps/_workspace/src/github.com/flynn/go-check"
	gzfs "github.com/flynn/flynn/Godeps/_workspace/src/github.com/mistifyio/go-zfs"
	"github.com/flynn/flynn/pkg/random"
)

// note: whimsical/unique dataset names per test are chosen to help debug
// in the event of stateful catastrophe (zfs is quite capable of carrying state
// between tests!).

type ZpoolTests struct{}

var _ = Suite(&ZpoolTests{})

func (ZpoolTests) SetUpSuite(c *C) {
	// Skip all tests in this suite if not running as root.
	// Many zfs operations require root priviledges.
	skipIfNotRoot(c)
}

var one_gig = int64(math.Pow(2, float64(30)))

func (ZpoolTests) TestProviderRequestingNonexistentZpoolFails(c *C) {
	dataset := "testpool-starfish"
	provider, err := NewProvider(&ProviderConfig{
		DatasetName: dataset,
		// no spec for making something, so this should be an error
	})
	c.Assert(provider, IsNil)
	c.Assert(err, NotNil)
	c.Assert(isDatasetNotExistsError(err), Equals, true)
}

func (ZpoolTests) TestProviderAutomaticFileVdevZpoolCreation(c *C) {
	dataset := "testpool-dinosaur"

	// don't actually use ioutil.Tempfile;
	// we want to exerise the path where the file doesn't exist.
	backingFilePath := fmt.Sprintf("/tmp/zfs-%s", random.String(12))
	defer os.Remove(backingFilePath)

	provider, err := NewProvider(&ProviderConfig{
		DatasetName: dataset,
		Make: &MakeDev{
			BackingFilename: backingFilePath,
			Size:            one_gig,
		},
	})
	defer func() {
		pool, _ := gzfs.GetZpool(dataset)
		if pool != nil {
			pool.Destroy()
		}
	}()
	c.Assert(err, IsNil)
	c.Assert(provider, NotNil)

	// also, we shouldn't get any '/testpool' dir at root
	_, err = os.Stat(dataset)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (ZpoolTests) TestProviderExistingZpoolDetection(c *C) {
	dataset := "testpool-festival"

	backingFilePath := fmt.Sprintf("/tmp/zfs-%s", random.String(12))
	defer os.Remove(backingFilePath)

	provider, err := NewProvider(&ProviderConfig{
		DatasetName: dataset,
		Make: &MakeDev{
			BackingFilename: backingFilePath,
			Size:            one_gig,
		},
	})
	defer func() {
		pool, _ := gzfs.GetZpool(dataset)
		if pool != nil {
			pool.Destroy()
		}
	}()
	c.Assert(err, IsNil)
	c.Assert(provider, NotNil)

	// if we create another provider with the same dataset, it should
	// see the existing one and thus shouldn't hit the MakeDev path
	badFilePath := "/tmp/zfs-test-should-not-exist"
	provider, err = NewProvider(&ProviderConfig{
		DatasetName: dataset,
		Make: &MakeDev{
			BackingFilename: badFilePath,
			Size:            one_gig,
		},
	})
	c.Assert(err, IsNil)
	c.Assert(provider, NotNil)
	_, err = os.Stat(badFilePath)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
}
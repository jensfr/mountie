package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/image/v5/copy"
	ociarchive "github.com/containers/image/v5/oci/archive"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	"github.com/sirupsen/logrus"
)

func main() {
	if reexec.Init() {
		return
	}

	policy, err := signature.DefaultPolicy(nil)
	if err != nil {
		fmt.Println(err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		fmt.Println(err)
	}

	payloadImage := "docker://quay.io/isolatedcontainers/kata-operator-payload:4.6.0"
	srcRef, err := alltransports.ParseImageName(payloadImage)
	if err != nil {
		fmt.Println("Invalid source name of payload container image: " + payloadImage)
		fmt.Println(err)
		os.Exit(-1)
	}
	inputFile := "/tmp/kata-install/kata-image.tar"
	destRef, err := alltransports.ParseImageName("oci-archive:" + inputFile)
	if err != nil {
		fmt.Println("Invalid destination name")
		fmt.Println(err)
		os.Exit(-1)
	}

	sourceCtx := &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "jensfr",
			Password: "jefrAZ82",
		},
	}

	_, err = copy.Image(context.Background(), policyContext, destRef, srcRef,
		&copy.Options{SourceCtx: sourceCtx})
	if err != nil {
		fmt.Println("copying image failed")
		fmt.Println(err)
		os.Exit(-1)
	}

	storeOpts := storage.StoreOptions{
		RunRoot:   "/tmp/run_teststore",
		GraphRoot: "/tmp/lib_teststore",
	}

	runtime, err := libpod.NewRuntime(context.Background(), libpod.WithStorageConfig(storeOpts))
	if err != nil {
		logrus.Fatal("Error creating runtime", err)
		return
	}

	if runtime != nil {
		fmt.Println("I got a runtime instance!")
	}

	allContainers, err := runtime.GetAllContainers()
	if err != nil {
		fmt.Println("GetAllContainers failed")
	}
	fmt.Println(len(allContainers))
	images, err := runtime.ImageRuntime().GetImages()
	if len(images) > 0 {
		fmt.Println("found images no: ")
		fmt.Println(len(images))
	}

	//newImages, err := runtime.ImageRuntime().LoadAllImagesFromDockerArchive(context.TODO(), inputFile, runtime.ImageRuntime().SignaturePolicyPath, os.Stdout)
	imageSrcRef, err := ociarchive.NewReference(inputFile, "")
	newImages, err := runtime.ImageRuntime().LoadFromArchiveReference(context.TODO(), imageSrcRef,
		runtime.ImageRuntime().SignaturePolicyPath, os.Stdout)
	if err != nil {
		fmt.Println("Loading images failed:")
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Print("loaded images:")
	fmt.Println(len(newImages))
	fmt.Println(newImages[0].ID())
	isMounted, existMountPath, err := newImages[0].Mounted()
	if err != nil {
		fmt.Println("get mount status failed")
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Println(existMountPath)
	if isMounted {
		fmt.Println("image is already mounted")
		os.Exit(-1)
	}

	/* First parameter to mount is options
	* The mount function calls lower-level mount functions
	* mountOptions from storage/drivers/driver.go:
	* `type MountOpts struct {
		// Mount label is the MAC Labels to assign to mount point (SELINUX)
		MountLabel string
		// UidMaps & GidMaps are the User Namespace mappings to be assigned to content in the mount point
		UidMaps []idtools.IDMap // nolint: golint
		GidMaps []idtools.IDMap // nolint: golint
		Options []string
		}`
	* Not sure what else can go into 'Options' here. Is there a way to
	* make the root mountpoint a known location instead of depending on the
	* new Image ID that's created during Load()?
	* Update: in storage/drivers/overlay/mount.go is:
	    options := &mountOptions{
		  Device: device,
		  Target: target,
		  Type:   mType,
		  Flag:   uint32(flags),
		  Label:  label,
		}
	* Another idea is to create a symlink /opt/kata that points to the created location
	* in the storage dir
	*/
	mountPath, err := newImages[0].Mount([]string{""}, "")
	if err != nil {
		fmt.Println("error mounting image: ")
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Println("Mountpath is:" + mountPath)
	umountErr := newImages[0].Unmount(false)
	if umountErr != nil {
		fmt.Println("error unmounting image: ")
		fmt.Println(umountErr)
		os.Exit(-1)
	}

}

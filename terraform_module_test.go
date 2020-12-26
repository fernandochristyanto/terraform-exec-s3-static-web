package test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
	"github.com/rs/xid"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestTerraformModule(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tfinstall")

	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpDir)

	ctx := context.Background()

	// Download latest version of terraform binary
	execPath, err := tfinstall.Find(ctx, tfinstall.LatestVersion(tmpDir, false))
	if err != nil {
		t.Error(err)
	}

	// Read configuration from ./testfixtures
	workingDir := "./testfixtures"
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		t.Error(err)
	}

	// Initialize terraform
	err = tf.Init(ctx, tfexec.Upgrade(true), tfexec.LockTimeout("60s"))
	if err != nil {
		t.Error(err)
	}

	bucketName := fmt.Sprintf("bucket_name=%s", xid.New().String())

	// Ensure terraform destroy even if error occurs
	defer tf.Destroy(ctx, tfexec.Var(bucketName))

	// Terraform apply with variable
	err = tf.Apply(ctx, tfexec.Var(bucketName))
	if err != nil {
		t.Error(err)
	}

	state, err := tf.Show(context.Background())
	if err != nil {
		t.Error(err)
	}

	// Read output value (in this case, S3 website url)
	endpoint := state.Values.Outputs["endpoint"].Value.(string)
	url := fmt.Sprintf("http://%s", endpoint)
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
	}

	// Web health check, fail test if status is not 200
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	t.Logf("\n%s", buf.String())

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code did not return 200")
	}
}

package pkg

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/pkg/types"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// PutClusterObject exports a cluster definition as json and places it in the target object storage bucket
func PutClusterObject(cr *types.StateStoreCredentials, d *types.StateStoreDetails, obj *types.PushBucketObject) error {
	ctx := context.Background()

	// Initialize minio client
	minioClient, err := minio.New(d.Hostname, &minio.Options{
		Creds:  credentials.NewStaticV4(cr.AccessKeyID, cr.SecretAccessKey, cr.SessionToken),
		Secure: true,
	})
	if err != nil {
		return fmt.Errorf("error initializing minio client: %s", err)
	}

	// Reference for cluster object output file
	object, err := os.Open(obj.LocalFilePath)
	if err != nil {
		return fmt.Errorf("error during object local copy file lookup: %s", err)
	}
	defer object.Close()

	objectStat, err := object.Stat()
	if err != nil {
		return fmt.Errorf("error during object stat: %s", err)
	}

	// Put
	_, err = minioClient.PutObject(
		ctx,
		d.Name,
		obj.RemoteFilePath,
		object,
		objectStat.Size(),
		minio.PutObjectOptions{ContentType: obj.ContentType},
	)
	if err != nil {
		return fmt.Errorf("error during object put: %s", err)
	}
	log.Info().Msgf("uploaded cluster object %s to state store bucket %s successfully", obj.LocalFilePath, d.Name)

	err = os.Remove(obj.LocalFilePath)
	if err != nil {
		return err
	}

	return nil
}

// ExportCluster proxy to kubefirst api /cluster/import to restore the database
func ExportCluster(kcfg k8s.KubernetesClient, cl types.Cluster) error {
	time.Sleep(time.Second * 10)

	err := pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", pkg.KubefirstConsoleLocalURLCloud), "kubefirst api")
	if err != nil {
		log.Error().Err(err).Msg("unable to start kubefirst api")
	}

	requestObject := types.ProxyImportRequest{
		Body: cl,
		Url:  "/cluster/import",
	}

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", pkg.KubefirstConsoleLocalURLCloud), bytes.NewReader(payload))
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to import cluster %s", res.Status)
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Info().Msgf("Import: %s", string(body))

	return nil
}

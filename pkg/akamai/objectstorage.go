package akamai

// GetAccessCredentials creates object store access credentials if they do not exist and returns them if they do
// func (c *AkamaiConfiguration) GetAccessCredentials(credentialName string, region string) (linodego.ObjectStorageKeyBucketAccess, error) {
func (c *AkamaiConfiguration) GetAccessCredentials(credentialName string, region string) (string, error) {

	// creds, err := c.checkKubefirstCredentials(credentialName, region)
	// if err != nil {
	// 	log.Info().Msg(err.Error())
	// }

	// if creds == (civogo.ObjectStoreCredential{}) {
	// 	log.Info().Msgf("credential name: %s not found, creating", credentialName)
	// 	creds, err = c.createAccessCredentials(credentialName, region)
	// 	if err != nil {
	// 		return civogo.ObjectStoreCredential{}, err
	// 	}

	// 	for i := 0; i < 12; i++ {
	// 		creds, err = c.getAccessCredentials(creds.ID, region)
	// 		if err != nil {
	// 			return civogo.ObjectStoreCredential{}, err
	// 		}
	// 		if creds.AccessKeyID != "" && creds.ID != "" && creds.Name != "" && creds.SecretAccessKeyID != "" {
	// 			break
	// 		}
	// 		log.Warn().Msg("waiting for civo credentials creation")
	// 		time.Sleep(time.Second * 10)
	// 	}

	// 	if creds.AccessKeyID == "" || creds.ID == "" || creds.Name == "" || creds.SecretAccessKeyID == "" {
	// 		log.Error().Msg("Civo credentials for state bucket in object storage could not be fetched, please try to run again")
	// 		os.Exit(1)
	// 	}
	// 	log.Info().Msgf("created object storage credential %s", credentialName)

	// 	return creds, nil
	// }

	return "creds", nil
}

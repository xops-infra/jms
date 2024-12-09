package io

import (
	"encoding/base64"
	"fmt"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
)

type KeyIO struct {
	db *db.DBService
}

func NewKey(db *db.DBService) *KeyIO {
	return &KeyIO{
		db: db,
	}
}

// getSignerByKeyID
func (k *KeyIO) GetSignerByKeyID(keyID string) (gossh.Signer, error) {
	keys, err := k.db.InternalLoadKey()
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		log.Debugf("keyid: %s key: %s", keyID, tea.Prettify(key))
		if *key.KeyID == keyID {
			return getSignerFromBase64(*key.PemBase64)
		}
	}
	return nil, fmt.Errorf("key not found: %s", keyID)
}

// getSignerByIdentityFile
func (k *KeyIO) GetSignerByIdentityFile(identityFile string) (gossh.Signer, error) {
	keys, err := k.db.InternalLoadKey()
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		if *key.IdentityFile == identityFile {
			return getSignerFromBase64(*key.PemBase64)
		}
	}
	return nil, fmt.Errorf("key not found: %s", identityFile)
}

func getSignerFromBase64(key string) (gossh.Signer, error) {
	// bas64 decode
	base64Pem, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	signer, err := gossh.ParsePrivateKey([]byte(base64Pem))
	if err != nil {
		return nil, err
	}
	return signer, nil
}

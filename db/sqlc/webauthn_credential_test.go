package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createCredentialForUser(t *testing.T, userID int64) WebauthnCredential {
	aaguid := util.RandomByteArray(16)

	transports := []protocol.AuthenticatorTransport{
		protocol.USB,
		protocol.NFC,
	}

	transportsJson, err := json.Marshal(transports)
	require.NoError(t, err)

	arg := CreateWebauthnCredentialsParams{
		ID:                      util.RandomByteArray(32),
		UserID:                  userID,
		PublicKey:               util.RandomByteArray(32),
		Transports:              transportsJson,
		UserPresent:             true,
		UserVerified:            true,
		BackupEligible:          true,
		BackupState:             true,
		Aaguid:                  uuid.UUID(aaguid),
		CloneWarning:            false,
		AuthenticatorAttachment: AuthenticatorAttachment(protocol.Platform),
		AuthenticatorData:       []byte{},
		PublicKeyAlgorithm:      -7,
	}

	cred, err := testStore.CreateWebauthnCredentials(context.Background(), arg)
	require.NoError(t, err)

	return cred
}

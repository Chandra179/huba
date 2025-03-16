package webauthn

import (
	"encoding/binary"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// User represents the user model for WebAuthn
type User struct {
	ID                        []byte
	Name                      string
	DisplayName               string
	Credentials               []webauthn.Credential
	RegistrationSessionData   *webauthn.SessionData
	AuthenticationSessionData *webauthn.SessionData
}

// NewUser creates a new User
func NewUser(name string, displayName string) *User {
	user := &User{
		ID:          make([]byte, 8),
		Name:        name,
		DisplayName: displayName,
		Credentials: []webauthn.Credential{},
	}

	// Generate a random ID
	uid, err := uuid.NewRandom()
	if err != nil {
		// If we can't generate a random UUID, use a simple counter
		binary.BigEndian.PutUint64(user.ID, uint64(1))
	} else {
		copy(user.ID, uid[:])
	}

	return user
}

// WebAuthnID returns the user's ID
func (u *User) WebAuthnID() []byte {
	return u.ID
}

// WebAuthnName returns the user's username
func (u *User) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the user's display name
func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnIcon returns the user's icon
func (u *User) WebAuthnIcon() string {
	return ""
}

// WebAuthnCredentials returns the user's credentials
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// AddCredential adds a credential to the user
func (u *User) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

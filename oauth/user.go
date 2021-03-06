package oauth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	pass "github.com/RichardKnop/go-oauth2-server/password"
	"github.com/RichardKnop/go-oauth2-server/util"
)

var (
	// MinPasswordLength defines minimum password length
	MinPasswordLength = 6

	// ErrPasswordTooShort ...
	ErrPasswordTooShort = fmt.Errorf(
		"Password must be at least %d characters long",
		MinPasswordLength,
	)
	// ErrUserNotFound ...
	ErrUserNotFound = errors.New("User not found")
	// ErrInvalidUserPassword ...
	ErrInvalidUserPassword = errors.New("Invalid user password")
	// ErrCannotSetEmptyUsername ...
	ErrCannotSetEmptyUsername = errors.New("Cannot set empty username")
	// ErrUserPasswordNotSet ...
	ErrUserPasswordNotSet = errors.New("User password not set")
	// ErrUsernameTaken ...
	ErrUsernameTaken = errors.New("Username taken")
)

// UserExists returns true if user exists
func (s *Service) UserExists(username string) bool {
	_, err := s.FindUserByUsername(username)
	return err == nil
}

// FindUserByUsername looks up a user by username
func (s *Service) FindUserByUsername(username string) (*User, error) {
	// Usernames are case insensitive
	user := new(User)
	notFound := s.db.Where("username = LOWER(?)", username).
		First(user).RecordNotFound()

	// Not found
	if notFound {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateUser saves a new user to database
func (s *Service) CreateUser(roleID, username, password string) (*User, error) {
	return s.createUserCommon(s.db, roleID, username, password)
}

// CreateUserTx saves a new user to database using injected db object
func (s *Service) CreateUserTx(tx *gorm.DB, roleID, username, password string) (*User, error) {
	return s.createUserCommon(tx, roleID, username, password)
}

// SetPassword sets a user password
func (s *Service) SetPassword(user *User, password string) error {
	return s.setPasswordCommon(s.db, user, password)
}

// SetPasswordTx sets a user password in a transaction
func (s *Service) SetPasswordTx(tx *gorm.DB, user *User, password string) error {
	return s.setPasswordCommon(tx, user, password)
}

// AuthUser authenticates user
func (s *Service) AuthUser(username, password string) (*User, error) {
	// Fetch the user
	user, err := s.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}

	// Check that the password is set
	if !user.Password.Valid {
		return nil, ErrUserPasswordNotSet
	}

	// Verify the password
	if pass.VerifyPassword(user.Password.String, password) != nil {
		return nil, ErrInvalidUserPassword
	}

	return user, nil
}

// UpdateUsername ...
func (s *Service) UpdateUsername(user *User, username string) error {
	if username == "" {
		return ErrCannotSetEmptyUsername
	}

	return s.updateUsernameCommon(s.db, user, username)
}

// UpdateUsernameTx ...
func (s *Service) UpdateUsernameTx(tx *gorm.DB, user *User, username string) error {
	return s.updateUsernameCommon(tx, user, username)
}

func (s *Service) createUserCommon(db *gorm.DB, roleID, username, password string) (*User, error) {
	// Start with a user without a password
	user := &User{
		RoleID:   util.StringOrNull(roleID),
		Username: strings.ToLower(username),
		Password: util.StringOrNull(""),
	}

	// If the password is being set already, create a bcrypt hash
	if password != "" {
		if len(password) < MinPasswordLength {
			return nil, ErrPasswordTooShort
		}
		passwordHash, err := pass.HashPassword(password)
		if err != nil {
			return nil, err
		}
		user.Password = util.StringOrNull(string(passwordHash))
	}

	// Check the username is available
	if s.UserExists(user.Username) {
		return nil, ErrUsernameTaken
	}

	// Create the user
	if err := db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) setPasswordCommon(db *gorm.DB, user *User, password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	// Create a bcrypt hash
	passwordHash, err := pass.HashPassword(password)
	if err != nil {
		return err
	}

	// Set the password on the user object
	return db.Model(user).UpdateColumns(User{
		Password: util.StringOrNull(string(passwordHash)),
		Model:    gorm.Model{UpdatedAt: time.Now().UTC()},
	}).Error
}

func (s *Service) updateUsernameCommon(db *gorm.DB, user *User, username string) error {
	if username == "" {
		return ErrCannotSetEmptyUsername
	}
	return db.Model(user).UpdateColumn("username", strings.ToLower(username)).Error
}

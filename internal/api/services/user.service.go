package services

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/minio"
	"github.com/emoss08/trenova/internal/util/password"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/google/uuid"
)

// UserService provides methods for user-related operations, including authentication,
// profile management, and interfacing with Minio for file storage. It encapsulates
// the logic necessary to interact with the underlying database and Minio server.
//
// Fields:
//   - Client: A *ent.Client object for database operations related to users.
//   - Logger: A *zerolog.Logger object used for logging messages in the service.
//   - Minio: A *minio.Client object for handling file uploads to Minio storage.
type UserService struct {
	Client *ent.Client
	Logger *zerolog.Logger
	Minio  *minio.Client
}

// NewUserService initializes a new instance of UserService with the provided dependencies.
// This function is typically called during application startup to set up services
// that will be used throughout the application's lifecycle.
//
// Parameters:
//   - s: A pointer to an api.Server struct that contains dependencies like the database client,
//     logger, and Minio client.
//
// Returns:
//   - *UserService: A pointer to a newly created UserService instance.
//
// Usage:
//
//	userService := NewUserService(server)
func NewUserService(s *api.Server) *UserService {
	return &UserService{
		Client: s.Client,
		Logger: s.Logger,
		Minio:  s.Minio,
	}
}

// GetAuthenticatedUser retrieves a user by their UUID from the database along with their roles and permissions.
// This function is typically used to authenticate a user and load their full authorization details in one go.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - userID: UUID of the user to retrieve.
//
// Returns:
//   - *ent.User: Pointer to the User entity if found.
//   - error: Error object if an error occurs during the database query. Nil if no error occurs.
//
// Possible errors:
//   - database query errors: returned directly with no modifications.
func (r *UserService) GetAuthenticatedUser(ctx context.Context, userID uuid.UUID) (*ent.User, error) {
	u, err := r.Client.User.
		Query().
		WithRoles(func(q *ent.RoleQuery) {
			q.WithPermissions()
		}).
		Where(user.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// UploadProfilePicture handles the upload of a profile picture for a specified user. It first checks if the user exists,
// reads the provided file data, renames the file to ensure uniqueness, uploads it to Minio storage, and finally updates
// the user's profile with the URL of the newly uploaded image.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - profilePicture: File header that includes metadata about the multipart uploaded file.
//   - userID: UUID of the user who is uploading the profile picture.
//
// Returns:
//   - *ent.User: Updated user entity with the new profile picture URL.
//   - error: Error object if an error occurs during any step of the process. Nil if the operation is successful.
//
// Errors:
//   - User existence check failure: Returns an error if the user does not exist.
//   - File handling errors: Includes errors during file opening, reading, or uploading.
//   - Database update errors: Occurs if the profile picture URL cannot be updated in the user's profile.
func (r *UserService) UploadProfilePicture(ctx context.Context, profilePicture *multipart.FileHeader, userID uuid.UUID) (*ent.User, error) {
	// Check if the user exists
	if err := r.checkUserExistence(ctx, userID); err != nil {
		return nil, err
	}

	fileData, err := r.readFileData(profilePicture)
	if err != nil {
		return nil, err
	}

	objectName, err := r.renameProfilePicture(profilePicture, userID)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to rename profile picture")
		return nil, err
	}

	user, err := r.uploadAndSetProfileURL(ctx, userID, objectName, profilePicture.Header.Get("Content-Type"), fileData)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// checkUserExistence confirms the existence of a user in the database by their UUID.
//
// Parameters:
//   - ctx: Context used for database query execution, containing possible deadlines or cancellation signals.
//   - userID: UUID of the user to check.
//
// Returns:
//   - error: Returns nil if the user exists, an error otherwise (either database query error or user does not exist).
//
// Errors:
//   - database query errors: If querying the database fails.
//   - user does not exist: If no user corresponds to the provided UUID.
func (r *UserService) checkUserExistence(ctx context.Context, userID uuid.UUID) error {
	exists, err := r.Client.User.Query().Where(user.IDEQ(userID)).Exist(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to query user existence")
		return err
	}
	if !exists {
		return errors.New("user does not exist")
	}
	return nil
}

// readFileData opens and reads the data from a provided multipart file header representing an uploaded file.
//
// Parameters:
//   - profilePicture: The file header containing metadata about the multipart uploaded file.
//
// Returns:
//   - []byte: Byte slice containing the read file data.
//   - error: Error object if an error occurs during file opening or reading.
//
// Errors:
//   - file opening errors: If the file cannot be opened.
//   - file reading errors: If reading the file data fails.
func (r *UserService) readFileData(profilePicture *multipart.FileHeader) ([]byte, error) {
	file, err := profilePicture.Open()
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to open profile picture file")
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// uploadAndSetProfileURL uploads file data to Minio and updates the user's profile with the new image URL.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - userID: UUID of the user whose profile picture is being updated.
//   - objectName: The name/path of the object in Minio where the file will be stored.
//   - contentType: MIME type of the file being uploaded.
//   - fileData: Byte slice containing the file data to be uploaded.
//
// Returns:
//   - *ent.User: Updated user entity with the new profile picture URL.
//   - error: Error object if an error occurs during file upload or database update.
//
// Errors:
//   - file upload errors: If uploading the file to Minio fails.
//   - database update errors: If updating the user's profile in the database fails.
func (r *UserService) uploadAndSetProfileURL(ctx context.Context, userID uuid.UUID, objectName, contentType string, fileData []byte) (*ent.User, error) {
	ui, err := r.Minio.SaveFile(ctx, "user-profile-pics", objectName, contentType, fileData)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to upload profile picture")
		return nil, err
	}

	user, err := r.Client.User.UpdateOneID(userID).SetProfilePicURL(ui).Save(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to save profile picture URL")
		return nil, err
	}
	return user, err
}

// renameProfilePicture generates a new, unique object name for a user's profile picture based on a random UUID.
//
// Parameters:
//   - profilePicture: The file header of the profile picture to be renamed.
//   - userID: UUID of the user whose profile picture is being renamed.
//
// Returns:
//   - string: The new object name incorporating the user's UUID and a random filename.
//   - error: Error object if generating a random filename fails.
//
// Errors:
//   - random filename generation errors: If generating the UUID fails.
func (r *UserService) renameProfilePicture(profilePicture *multipart.FileHeader, userID uuid.UUID) (string, error) {
	randomFilename, err := uuid.NewRandom()
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to generate random filename")
		return "", err
	}

	fileExt := filepath.Ext(profilePicture.Filename)
	if fileExt == "" {
		return "", errors.New("failed to get file extension")
	}

	return userID.String() + "/" + randomFilename.String() + fileExt, nil
}

// ChangePassword updates a user's password in the database after verifying the old password.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - userID: UUID of the user whose password is being updated.
//   - oldPassword: The user's current password.
//   - newPassword: The new password to set for the user.
//
// Returns:
//   - error: Error object if an error occurs during the password change operation.
//
// Errors:
//   - user existence check failure: If the user does not exist.
func (r *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Check if the user exists
	if err := r.checkUserExistence(ctx, userID); err != nil {
		return err
	}

	// Get the user from the database
	u, err := r.Client.User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to query user")
		return err
	}

	// Check if the old password matches the user's current password
	if err = password.Verify(u.Password, oldPassword); err != nil {
		return err
	}

	// Hash the new password
	hashedPassword := password.Generate(newPassword)

	// Update the user's password
	_, err = r.Client.User.UpdateOneID(userID).SetPassword(hashedPassword).Save(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to update user password")
		return err
	}

	return nil
}

// UpdateUser updates a user's profile information in the database. It first retrieves the current user entity,
// checks if the version has changed, and then updates the user's profile with the new information.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - entity: The updated user entity to save in the database.
//
// Returns:
//   - *ent.User: Updated user entity with the new profile information.
//   - error: Error object if an error occurs during the database update operation.
func (r *UserService) UpdateUser(ctx context.Context, entity *ent.User) (*ent.User, error) {
	var updatedEntity *ent.User

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error

		updatedEntity, err = r.updateUserEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})

	return updatedEntity, err
}

// updateUserEntity updates a user's profile information in the database within a transaction. It first retrieves the
// current user entity, checks if the version has changed, and then updates the user's profile with the new information.
//
// Parameters:
//   - ctx: Context used for the operation, which may include deadlines or cancellation signals.
//   - tx: The transaction object used to execute the database update operation.
//   - entity: The updated user entity to save in the database.
//
// Returns:
//   - *ent.User: Updated user entity with the new profile information.
//   - error: Error object if an error occurs during the database update operation.
func (r *UserService) updateUserEntity(ctx context.Context, tx *ent.Tx, entity *ent.User) (*ent.User, error) {
	current, err := tx.User.Get(ctx, entity.ID)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to get user")
		return nil, err
	}

	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.User.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetEmail(entity.Email).
		SetUsername(entity.Username).
		SetTimezone(entity.Timezone).
		SetPhoneNumber(entity.PhoneNumber).
		SetVersion(entity.Version + 1) // Increment the version (optimistic locking)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to update user")
		return nil, err
	}

	return updatedEntity, nil
}

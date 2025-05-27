package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sonartest_cart/app/dto"
)

func TestSaveUserDetails(t *testing.T) {
	tests := []struct {
		name    string
		req     *dto.UserDetailSaveRequest
		wantID  int64
		wantErr bool
		query   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "success-case",
			req: &dto.UserDetailSaveRequest{
				UserID:   1,
				UserName: "johndoe",
				Password: "securepwd",
				Address:  "123 Street",
				Pincode:  123456,
				Phone:    9876543210,
				Mail:     "john@example.com",
				IsAdmin:  false,
			},
			wantID:  1,
			wantErr: false,
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// Match the exact SQL pattern GORM generates
				mock.ExpectQuery(`^INSERT INTO "userdetails" \("username","password","address","pincode","phone_number","mail","status","updated_at","isadmin","id"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8,\$9,\$10\) RETURNING "id"$`).
					WithArgs(
						"johndoe",          // Username (now first)
						"securepwd",        // Password
						"123 Street",       // Address
						int64(123456),      // Pincode
						int64(9876543210),  // Phone
						"john@example.com", // Mail
						true,               // Status
						sqlmock.AnyArg(),   // UpdatedAt
						false,              // IsAdmin
						1,                  // ID (now last)
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
		},
		{
			name: "failure-case",
			req: &dto.UserDetailSaveRequest{
				UserID:   2,
				UserName: "janedoe",
			},
			wantID:  0,
			wantErr: true,
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`^INSERT INTO "userdetails"`).
					WillReturnError(fmt.Errorf("insert failed"))
				mock.ExpectRollback()
			},
		},
	}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	repo := NewUserRepo(gdb)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.query(mock)

			gotID, err := repo.SaveUserDetails(test.req)
			if (err != nil) != test.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if gotID != test.wantID {
				t.Errorf("expected ID %d, got %d", test.wantID, gotID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantUser *Userdetail
		wantErr  bool
		query    func(mock sqlmock.Sqlmock)
	}{
		{
			name:     "success-case",
			username: "johndoe",
			wantUser: &Userdetail{
				ID:          1,
				Username:    "johndoe",
				Password:    "securepwd",
				Address:     "123 Street",
				Pincode:     123456,
				Phonenumber: 9876543210,
				Mail:        "john@example.com",
				Status:      true,
				UpdatedAt:   time.Now(),
				IsAdmin:     false,
			},
			wantErr: false,
			query: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "username", "password", "address", "pincode",
					"phone_number", "mail", "status", "updated_at", "isadmin",
				}).AddRow(
					1, "johndoe", "securepwd", "123 Street", 123456,
					9876543210, "john@example.com", true, time.Now(), false,
				)

				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE username = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs("johndoe", 1).
					WillReturnRows(rows)
			},
		},
		{
			name:     "not-found-case",
			username: "nonexistent",
			wantUser: nil,
			wantErr:  true,
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE username = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs("nonexistent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
		},
		{
			name:     "database-error-case",
			username: "johndoe",
			wantUser: nil,
			wantErr:  true,
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE username = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs("johndoe", 1).
					WillReturnError(fmt.Errorf("database error"))
			},
		},
	}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	repo := NewUserRepo(gdb)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.query(mock)

			gotUser, err := repo.GetUserByUsername(test.username)

			// Check error expectations first
			if (err != nil) != test.wantErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// For error cases, we don't need to check the user object
			if test.wantErr {
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Errorf("there were unfulfilled expectations: %s", err)
				}
				return
			}

			// For success cases, verify the user object
			if gotUser == nil {
				t.Error("expected user, got nil")
				return
			}

			if gotUser.ID != test.wantUser.ID {
				t.Errorf("expected ID %d, got %d", test.wantUser.ID, gotUser.ID)
			}
			if gotUser.Username != test.wantUser.Username {
				t.Errorf("expected Username %s, got %s", test.wantUser.Username, gotUser.Username)
			}
			// Add similar checks for other fields if needed

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestIsUserActive(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		want      bool
		wantErr   bool
		errString string
		query     func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "active-user",
			userID:  1,
			want:    true,
			wantErr: false,
			query: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "status"}).
					AddRow(1, true)
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE id = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs(int64(1), 1).
					WillReturnRows(rows)
			},
		},
		{
			name:    "inactive-user",
			userID:  2,
			want:    false,
			wantErr: false,
			query: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "status"}).
					AddRow(2, false)
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE id = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs(int64(2), 1).
					WillReturnRows(rows)
			},
		},
		{
			name:      "user-not-found",
			userID:    999,
			want:      false,
			wantErr:   true,
			errString: "user not found",
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE id = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs(int64(999), 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
		},
		{
			name:      "database-error",
			userID:    3,
			want:      false,
			wantErr:   true,
			errString: "",
			query: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`^SELECT \* FROM "userdetails" WHERE id = \$1 ORDER BY "userdetails"."id" LIMIT \$2$`).
					WithArgs(int64(3), 1).
					WillReturnError(fmt.Errorf("database error"))
			},
		},
	}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	repo := NewUserRepo(gdb)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.query(mock)

			got, err := repo.IsUserActive(test.userID)

			if (err != nil) != test.wantErr {
				t.Errorf("IsUserActive() error = %v, wantErr %v", err, test.wantErr)
				return
			}

			if test.wantErr {
				if test.errString != "" && (err == nil || !strings.Contains(err.Error(), test.errString)) {
					t.Errorf("IsUserActive() error = %v, want contains %q", err, test.errString)
				}
			} else {
				if got != test.want {
					t.Errorf("IsUserActive() = %v, want %v", got, test.want)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

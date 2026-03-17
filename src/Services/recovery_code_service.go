package Services

import (
	"crypto/rand"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const recoveryCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type RecoveryCodeResult struct {
	Id   int64  `json:"id"`
	Code string `json:"code"`
}

func generateRecoveryCode() (string, error) {
	segmentLen := 4
	segments := 4
	total := segmentLen * segments

	buf := make([]byte, total)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	for i := range buf {
		buf[i] = recoveryCodeChars[int(buf[i])%len(recoveryCodeChars)]
	}

	code := fmt.Sprintf("%s-%s-%s-%s",
		string(buf[0:4]),
		string(buf[4:8]),
		string(buf[8:12]),
		string(buf[12:16]),
	)
	return code, nil
}

// GenerateAndStoreRecoveryCodes generates 5 recovery codes, hashes them, stores them in the DB,
// and returns the plain-text codes with their IDs to be shown to the user once.
func GenerateAndStoreRecoveryCodes(userID int, db *sql.DB) ([]RecoveryCodeResult, error) {
	plain := make([]string, 5)
	for i := range plain {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		plain[i] = code
	}

	var results []RecoveryCodeResult
	for _, code := range plain {
		hash, err := bcrypt.GenerateFromPassword([]byte(code), 14)
		if err != nil {
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		res, err := db.Exec(
			"INSERT INTO recovery_codes (user_id, recovery_code) VALUES (?, ?)",
			userID, string(hash),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to store recovery code: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get recovery code ID: %w", err)
		}
		results = append(results, RecoveryCodeResult{Id: id, Code: code})
	}

	return results, nil
}

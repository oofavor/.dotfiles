package throne

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"harpo/internal/xray"
)

// LoadProfiles reads Xray VLESS profiles read-only from Throne's SQLite database.
func LoadProfiles(dbPath string) ([]xray.Profile, error) {
	if fi, err := os.Stat(dbPath); err != nil || fi.IsDir() {
		return nil, nil
	}
	dsn := fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(2000)", filepath.ToSlash(dbPath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, type, outbound_json FROM profiles WHERE type = 'xrayvless' ORDER BY gid, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []xray.Profile
	for rows.Next() {
		var (
			id    int64
			name  sql.NullString
			ptype sql.NullString
			oj    sql.NullString
		)
		if err := rows.Scan(&id, &name, &ptype, &oj); err != nil {
			continue
		}
		if !oj.Valid || oj.String == "" {
			continue
		}
		var ob map[string]any
		if err := json.Unmarshal([]byte(oj.String), &ob); err != nil {
			continue
		}
		profiles = append(profiles, xray.ProfileFromOutbound(name.String, "Throne DB", ob))
	}
	return profiles, rows.Err()
}

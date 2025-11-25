package main

import (
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
)

// CloseDBConnection güncellendi
func CloseDBConnection() {
	if g.QCM != nil && g.QCM.DB != nil {
		g.QCM.DB.Close()
		clog(LevelInfo, "SQL Server bağlantısı kapatıldı.")
	}
}

func getAllServersWithType(db *sql.DB, serverType int) (map[int64]*WbpServer, error) {
	query := `
		SELECT 
			id, ip, domain, properties, enabled, deleted, user_count, 
			type, name, time_zone, in_use, master, cps_limit, max_lines, 
			refresh_strategy, ari_url, ari_username, ari_password
		FROM dbo.wbp_server WHERE  [type] = @type and [enabled] = 1 
	`

	rows, err := db.Query(query, sql.Named("type", serverType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make(map[int64]*WbpServer)

	for rows.Next() {
		var s WbpServer
		// Scan sırası SELECT sorgusundaki sırayla birebir aynı olmalıdır
		err := rows.Scan(
			&s.ID,
			&s.IP,
			&s.Domain,
			&s.Properties,
			&s.Enabled,
			&s.Deleted,
			&s.UserCount,
			&s.Type,
			&s.Name,
			&s.TimeZone,
			&s.InUse,
			&s.Master,
			&s.CpsLimit,
			&s.MaxLines,
			&s.RefreshStrategy,
			&s.AriURL,
			&s.AriUsername,
			&s.AriPassword,
		)
		if err != nil {
			return nil, err
		}
		servers[s.ID] = &s
	}

	return servers, nil
}

// getServer verilen ID'ye sahip sunucu bilgisini döner.
// Eğer kayıt bulunamazsa hata döner.
func getServer(db *sql.DB, id int64) (*WbpServer, error) {
	query := `
		SELECT 
			id, ip, domain, properties, enabled, deleted, user_count, 
			type, name, time_zone, in_use, master, cps_limit, max_lines, 
			refresh_strategy, ari_url, ari_username, ari_password
		FROM dbo.wbp_server
		WHERE id = @p1
	`

	var s WbpServer

	// QueryRow tek bir satır beklediğimiz sorgularda kullanılır.
	// id parametresi sorgudaki @p1 yerine geçer.
	row := db.QueryRow(query, id)

	err := row.Scan(
		&s.ID,
		&s.IP,
		&s.Domain,
		&s.Properties,
		&s.Enabled,
		&s.Deleted,
		&s.UserCount,
		&s.Type,
		&s.Name,
		&s.TimeZone,
		&s.InUse,
		&s.Master,
		&s.CpsLimit,
		&s.MaxLines,
		&s.RefreshStrategy,
		&s.AriURL,
		&s.AriUsername,
		&s.AriPassword,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Kayıt bulunamadı durumu
			return nil, fmt.Errorf("ID %d ile kayıtlı sunucu bulunamadı", id)
		}
		// Diğer veritabanı hataları
		return nil, err
	}

	return &s, nil
}

// Tüm konfigürasyonları liste olarak çeker
func getAllAppConfigs(db *sql.DB) ([]QeAppConfig, error) {
	query := "SELECT prop_key, prop_value FROM dbo.qe_app_config"

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []QeAppConfig

	for rows.Next() {
		var c QeAppConfig
		if err := rows.Scan(&c.PropKey, &c.PropValue); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}

	return configs, nil
}

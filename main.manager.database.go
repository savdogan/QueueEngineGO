package main

import (
	_ "github.com/microsoft/go-mssqldb"
)

// CloseDBConnection güncellendi
func CloseDBConnection() {
	if g.QCM != nil && g.QCM.DB != nil {
		g.QCM.DB.Close()
		clog(LevelInfo, "SQL Server bağlantısı kapatıldı.")
	}
}

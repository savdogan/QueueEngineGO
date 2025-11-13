package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		// Hata oluşursa, logla ve varsayılan bir değer döndür
		fmt.Printf("Hostname alınamadı: %v\n", err)
		return "UNKNOWN_HOST"
	}
	return name
}

func NullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Integer
func NullInt64ToInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// Float
func NullFloat64ToFloat64(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}

// Boolean
func NullBoolToBool(nb sql.NullBool) bool {
	if nb.Valid {
		return nb.Bool
	}
	return false
}

// Time
func NullTimeToTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{} // "zero time" döner
}

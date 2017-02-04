package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
)

type reportHandler struct {
	db *sql.DB
}

func (h *reportHandler) Stats(w http.ResponseWriter, r *http.Request) {
	var count int
	row := h.db.QueryRow("SELECT count(*) FROM results")
	if err := row.Scan(&count); err != nil {
		http.Error(w, "can't get results", 500)
		return
	}
	writeJSON(w, map[string]string{"count": fmt.Sprint(count)})
}

func (h *reportHandler) RegisterCall(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	location := r.FormValue("location")
	if location == "" {
		location = "NONE"
	}
	result := r.FormValue("result")
	if result == "" {
		http.Error(w, "must pass result", 400)
		return
	}
	contactID := r.FormValue("contactid")
	if contactID == "" {
		http.Error(w, "must pass contact id", 400)
		return
	}
	issueID := r.FormValue("issueid")
	if issueID == "" {
		http.Error(w, "must pass issue id", 400)
		return
	}
	phone := r.FormValue("phone") // "" means no phone specified -- either old data or old client

	stmt, err := h.db.Prepare("INSERT INTO results(location, result, time, issueID, contactID, phone) values(?,?,?,?,?,?)")
	if err != nil {
		http.Error(w, "can't prepare statement", 500)
		return
	}
	defer stmt.Close()

	now := time.Now()
	_, err = stmt.Exec(location, result, now.Unix(), issueID, contactID, phone)
	if err != nil {
		http.Error(w, "can't exec statement", 500)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (h *reportHandler) MigrateDB() error {
	rows, err := h.db.Query("PRAGMA table_info(results)")
	if err != nil {
		return err
	}
	defer rows.Close()
	hasphone := false
	for rows.Next() {
		var (
			cidx    int            // column index
			name    string         // column name
			typ     string         // data type
			nonnull int            // has nonnull constraint
			def     sql.NullString // default value
			pk      int            // is part of the primary key
		)
		if err := rows.Scan(&cidx, &name, &typ, &nonnull, &def, &pk); err != nil {
			return err
		}
		if name == "phone" {
			hasphone = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasphone {
		log.Printf("running migration: add phone column to results table")
		_, err := h.db.Exec("ALTER TABLE results ADD COLUMN phone TEXT")
		if err != nil {
			return err
		}
	}
	return nil
}

package sqlite

//
//import (
//	"URL-Shortener/internal/storage"
//	"database/sql"
//	"errors"
//	"fmt"
//	"github.com/mattn/go-sqlite3"
//)
//
//type Storage struct {
//	db *sql.DB
//}
//
//func NewStorage(storagePath string) (*Storage, error) {
//	const op = "storage.sqlite.NewStorage"
//
//	db, err := sql.Open("sqlite3", storagePath)
//	if err != nil {
//		return nil, fmt.Errorf("%s: %w", op, err)
//	}
//
//	if err := db.Ping(); err != nil {
//		return nil, fmt.Errorf("%s: ping: %w", op, err)
//	}
//
//	stmt, err := db.Prepare(`
//		CREATE TABLE IF NOT EXISTS URLs (
//		    id INTEGER PRIMARY KEY,
//		    alias TEXT NOT NULL UNIQUE,
//		    url TEXT NOT NULL);
//		CREATE INDEX IF NOT EXISTS idx_alias ON URLs(alias);
//`)
//	if err != nil {
//		return nil, fmt.Errorf("%s: %w", op, err)
//	}
//	_, err = stmt.Exec()
//	if err != nil {
//		return nil, fmt.Errorf("%s: %w", op, err)
//	}
//	return &Storage{db: db}, nil
//}
//
//func (s *Storage) Close() error {
//	return s.db.Close()
//}
//
//func (s *Storage) SaveURL(alias string, urlToSave string) (int64, error) {
//	const op = "storage.sqlite.SaveURL"
//
//	stmt, err := s.db.Prepare("INSERT INTO URLs(alias, url) VALUES(?, ?)")
//	if err != nil {
//		return 0, fmt.Errorf("%s: %w", op, err)
//	}
//	defer stmt.Close()
//
//	res, err := stmt.Exec(alias, urlToSave)
//	if err != nil {
//		var sqliteErr sqlite3.Error
//		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
//			return 0, fmt.Errorf("%s: %w", op, storage.ErrAliasExists)
//		}
//		return 0, fmt.Errorf("%s: %w", op, err)
//	}
//
//	id, err := res.LastInsertId()
//	if err != nil {
//		return 0, fmt.Errorf("%s: get last insert id: %w", op, err)
//	}
//
//	return id, nil
//}
//
//func (s *Storage) GetUrl(alias string) (string, error) {
//	const op = "storage.sqlite.GetUrl"
//
//	stmt, err := s.db.Prepare("SELECT url FROM URLs WHERE alias = ?")
//	if err != nil {
//		return "", fmt.Errorf("%s: %w", op, err)
//	}
//	defer stmt.Close()
//
//	var url string
//	err = stmt.QueryRow(alias).Scan(&url)
//	if err != nil {
//		if errors.Is(err, sql.ErrNoRows) {
//			return "", storage.ErrUrlNotFound
//		}
//		return "", fmt.Errorf("%s: execute statement: %w", op, err)
//	}
//
//	return url, nil
//}
//
//func (s *Storage) DeleteUrl(alias string) error {
//	const op = "storage.sqlite.DeleteUrl"
//
//	result, err := s.db.Exec("DELETE FROM URLs WHERE alias = ?", alias)
//	if err != nil {
//		return fmt.Errorf("%s: %w", op, err)
//	}
//
//	rowsAffected, err := result.RowsAffected()
//	if err != nil {
//		return fmt.Errorf("%s: get rows affected: %w", op, err)
//	}
//
//	if rowsAffected == 0 {
//		return storage.ErrUrlNotFound
//	}
//
//	return nil
//}
//
//func (s *Storage) AliasExists(alias string) (bool, error) {
//	const op = "storage.sqlite.AliasExists"
//
//	var exists bool
//	err := s.db.QueryRow("SELECT EXISTS (SELECT 1 FROM URLs WHERE alias = ?)", alias).Scan(&exists)
//	if err != nil {
//		return false, fmt.Errorf("%s: %w", op, err)
//	}
//
//	return exists, nil
//}

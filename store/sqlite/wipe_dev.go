//go:build dev

package sqlite

import "log"

func (d DB) Clear() {
	log.Printf("clearing all SQLite tables")
	if _, err := d.ctx.Exec(`DELETE FROM entries`); err != nil {
		log.Fatalf("failed to delete movies: %v", err)
	}
	if _, err := d.ctx.Exec(`DELETE FROM guest_links`); err != nil {
		log.Fatalf("failed to delete reviews: %v", err)
	}
	if _, err := d.ctx.Exec(`DELETE FROM settings`); err != nil {
		log.Fatalf("failed to delete users: %v", err)
	}
}

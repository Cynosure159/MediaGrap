package metadata

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func TestPreparedNFOAtomicSavedPriority(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "metadata.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	media := filepath.Join(root, "Movie.mkv")
	if err := os.WriteFile(filepath.Join(root, "movie.nfo"), []byte(`<movie><title>Local</title><plot>Local plot</plot></movie>`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'Test',?)`, root); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, nil)
	for i, provider := range []string{"manual", "tmdb", "nfo"} {
		id := int64(i + 1)
		if _, err := db.Exec(`INSERT INTO media_items(id,source_id,relative_path,title_hint,file_size,modified_at) VALUES(?,1,?,'Movie',1,'now')`, id, provider+".mkv"); err != nil {
			t.Fatal(err)
		}
		prepared, found, err := s.PrepareExistingNFO(t.Context(), id, media)
		if err != nil || !found {
			t.Fatal(found, err)
		}
		saved := Record{MediaItemID: id, Provider: provider, Title: "Saved", Overview: "Keep", LockedFields: []string{"title"}}
		if _, err := s.Save(t.Context(), saved); err != nil {
			t.Fatal(err)
		}
		if err := s.HydratePreparedNFO(t.Context(), prepared); err != nil {
			t.Fatal(err)
		}
		record, err := s.Record(t.Context(), id)
		if err != nil || record.Title != "Saved" || record.Overview != "Keep" || record.Provider != provider || len(record.LockedFields) != 1 {
			t.Fatalf("%+v %v", record, err)
		}
		// Whichever mutation wins first, a concurrent manual save must win last:
		// hydration is conditional, but the ordinary user save is unconditional.
		for n := 0; n < 20; n++ {
			if _, err := db.Exec(`DELETE FROM media_metadata WHERE media_item_id=?`, id); err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			errs := make(chan error, 2)
			start := make(chan struct{})
			wg.Add(2)
			go func() { defer wg.Done(); <-start; errs <- s.HydratePreparedNFO(t.Context(), prepared) }()
			go func() { defer wg.Done(); <-start; _, err := s.Save(t.Context(), saved); errs <- err }()
			close(start)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					t.Fatal(err)
				}
			}
			record, err = s.Record(t.Context(), id)
			if err != nil || record.Title != "Saved" || record.Provider != provider {
				t.Fatalf("%+v %v", record, err)
			}
		}
	}
}

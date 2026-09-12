package jobs

import "testing"

func TestActiveScanPagesAndExactTerminalBeyondHistory(t *testing.T) {
	s := newTestJobsService(t)
	for i := 1; i <= 205; i++ {
		if _, err := s.db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(?,'Fixture',?)`, i, i); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec(`INSERT INTO jobs(id,source_id,kind,state,payload) VALUES(?,?,'scan','running','{"secret":"hidden"}')`, i, i); err != nil {
			t.Fatal(err)
		}
	}
	for i := 206; i <= 410; i++ {
		if _, err := s.db.Exec(`INSERT INTO jobs(id,kind,state) VALUES(?,'other','succeeded')`, i); err != nil {
			t.Fatal(err)
		}
	}
	recent, err := s.List(t.Context())
	if err != nil || len(recent) != 100 || recent[0].ID != 410 {
		t.Fatal(len(recent), err)
	}
	cursor := int64(0)
	count := 0
	for _, expected := range []int{100, 100, 5} {
		page, err := s.ActiveScans(t.Context(), cursor)
		if err != nil || len(page) != expected {
			t.Fatal(len(page), err)
		}
		for _, job := range page {
			if job.ID <= cursor || job.Kind != "scan" || job.State != "running" {
				t.Fatal(job)
			}
			cursor = job.ID
			count++
		}
	}
	if count != 205 {
		t.Fatal(count)
	}
	if _, err := s.db.Exec(`UPDATE jobs SET state='succeeded' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	terminal, err := s.Get(t.Context(), 1)
	if err != nil || terminal.State != "succeeded" {
		t.Fatal(terminal, err)
	}
	page, err := s.ActiveScans(t.Context(), 0)
	if err != nil || page[0].ID != 2 {
		t.Fatal(page, err)
	}
}

package db

import (
	"context"
	"fmt"
	"log"
	"time"

	couchdb "github.com/leporo/go-couchdb"
	"ntust-alumni/backend/internal/models"
)

type CouchDB struct {
	client *couchdb.CouchDB
}

func NewCouchDB(url string) (*CouchDB, error) {
	client, err := couchdb.NewCouchDB(url)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client couchdb: %w", err)
	}

	// Uji koneksi
	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke couchdb: %w", err)
	}

	log.Println("Terhubung ke CouchDB")

	return &CouchDB{client: client}, nil
}

// ensureDatabase memastikan database ada, jika tidak maka buat
func (c *CouchDB) ensureDatabase(ctx context.Context, name string) error {
	dbs, err := c.client.ListDBs()
	if err != nil {
		return fmt.Errorf("gagal list database: %w", err)
	}

	for _, db := range dbs {
		if db == name {
			return nil
		}
	}

	if err := c.client.CreateDB(name); err != nil {
		return fmt.Errorf("gagal membuat database %s: %w", name, err)
	}
	log.Printf("Database %s dibuat", name)
	return nil
}

// InitDatabase inisialisasi semua database yang diperlukan
func (c *CouchDB) InitDatabase(ctx context.Context) error {
	for _, name := range []string{"alumni", "subgroups", "forum_posts"} {
		if err := c.ensureDatabase(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// --- Alumni ---

func (c *CouchDB) GetAlumni(ctx context.Context, id string) (*models.Alumni, error) {
	var a models.Alumni
	if err := c.client.Get(ctx, "alumni", id, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *CouchDB) SaveAlumni(ctx context.Context, a *models.Alumni) error {
	now := time.Now()
	if a.DibuatPada.IsZero() {
		a.DibuatPada = now
	}
	a.Diperbarui = now

	resp, err := c.client.Post(ctx, "alumni", a)
	if err != nil {
		return err
	}
	a.ID = resp.ID
	a.Rev = resp.Rev
	return nil
}

func (c *CouchDB) UpdateAlumni(ctx context.Context, a *models.Alumni) error {
	a.Diperbarui = time.Now()
	_, err := c.client.Put(ctx, "alumni", a.ID, a.Rev, a)
	return err
}

func (c *CouchDB) ListAlumni(ctx context.Context) ([]models.Alumni, error) {
	var result struct {
		Rows []struct {
			Doc models.Alumni `json:"doc"`
		} `json:"rows"`
	}
	if err := c.client.Get(ctx, "alumni/_all_docs?include_docs=true", &result); err != nil {
		return nil, err
	}
	var alumni []models.Alumni
	for _, row := range result.Rows {
		alumni = append(alumni, row.Doc)
	}
	return alumni, nil
}

func (c *CouchDB) FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error) {
	query := map[string]interface{}{
		"selector": map[string]interface{}{
			"email": email,
		},
		"fields": []interface{}{"_id", "_rev", "nama", "email", "nim", "jurusan", "tahun_lulus", "wilayah", "telepon", "role", "subgroup_id", "dibuat_pada", "diperbarui"},
	}
	var result struct {
		Docs []models.Alumni `json:"docs"`
	}
	if err := c.client.Post(ctx, "alumni/_find", query, &result); err != nil {
		return nil, err
	}
	if len(result.Docs) == 0 {
		return nil, nil
	}
	return &result.Docs[0], nil
}

// --- Subgroup ---

func (c *CouchDB) GetSubgroup(ctx context.Context, id string) (*models.Subgroup, error) {
	var s models.Subgroup
	if err := c.client.Get(ctx, "subgroups", id, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *CouchDB) SaveSubgroup(ctx context.Context, s *models.Subgroup) error {
	now := time.Now()
	if s.DibuatPada.IsZero() {
		s.DibuatPada = now
	}
	s.Diperbarui = now

	resp, err := c.client.Post(ctx, "subgroups", s)
	if err != nil {
		return err
	}
	s.ID = resp.ID
	s.Rev = resp.Rev
	return nil
}

func (c *CouchDB) UpdateSubgroup(ctx context.Context, s *models.Subgroup) error {
	s.Diperbarui = time.Now()
	_, err := c.client.Put(ctx, "subgroups", s.ID, s.Rev, s)
	return err
}

func (c *CouchDB) DeleteSubgroup(ctx context.Context, id, rev string) error {
	_, err := c.client.Delete(ctx, "subgroups", id, rev)
	return err
}

func (c *CouchDB) ListSubgroups(ctx context.Context) ([]models.Subgroup, error) {
	var result struct {
		Rows []struct {
			Doc models.Subgroup `json:"doc"`
		} `json:"rows"`
	}
	if err := c.client.Get(ctx, "subgroups/_all_docs?include_docs=true", &result); err != nil {
		return nil, err
	}
	var subgroups []models.Subgroup
	for _, row := range result.Rows {
		subgroups = append(subgroups, row.Doc)
	}
	return subgroups, nil
}

// --- Forum Post ---

func (c *CouchDB) SaveForumPost(ctx context.Context, p *models.ForumPost) error {
	now := time.Now()
	if p.DibuatPada.IsZero() {
		p.DibuatPada = now
	}
	p.Diperbarui = now

	resp, err := c.client.Post(ctx, "forum_posts", p)
	if err != nil {
		return err
	}
	p.ID = resp.ID
	p.Rev = resp.Rev
	return nil
}

func (c *CouchDB) ListForumPosts(ctx context.Context) ([]models.ForumPost, error) {
	var result struct {
		Rows []struct {
			Doc models.ForumPost `json:"doc"`
		} `json:"rows"`
	}
	if err := c.client.Get(ctx, "forum_posts/_all_docs?include_docs=true&descending=true", &result); err != nil {
		return nil, err
	}
	var posts []models.ForumPost
	for _, row := range result.Rows {
		posts = append(posts, row.Doc)
	}
	return posts, nil
}

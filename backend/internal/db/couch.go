package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"ntust-alumni/backend/internal/models"
)

// Store adalah interface untuk operasi database alumni, subgroup, dan forum post.
type Store interface {
	GetAlumni(ctx context.Context, id string) (*models.Alumni, error)
	SaveAlumni(ctx context.Context, a *models.Alumni) error
	UpdateAlumni(ctx context.Context, a *models.Alumni) error
	ListAlumni(ctx context.Context) ([]models.Alumni, error)
	FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error)

	GetSubgroup(ctx context.Context, id string) (*models.Subgroup, error)
	SaveSubgroup(ctx context.Context, s *models.Subgroup) error
	UpdateSubgroup(ctx context.Context, s *models.Subgroup) error
	DeleteSubgroup(ctx context.Context, id, rev string) error
	ListSubgroups(ctx context.Context) ([]models.Subgroup, error)

	SaveForumPost(ctx context.Context, p *models.ForumPost) error
	ListForumPosts(ctx context.Context) ([]models.ForumPost, error)

	FindAlumniByVerificationToken(ctx context.Context, token string) (*models.Alumni, error)

	InitDatabase(ctx context.Context) error
	Ping() error
}

var _ Store = (*CouchDB)(nil)

type CouchDB struct {
	baseURL string
	client  *http.Client
}

func NewCouchDB(url string) (*CouchDB, error) {
	c := &CouchDB{
		baseURL: url,
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	if err := c.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke couchdb: %w", err)
	}

	log.Println("Terhubung ke CouchDB")
	return c, nil
}

func (c *CouchDB) Ping() error {
	resp, err := c.client.Get(c.baseURL + "/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("couchdb responded with status %d", resp.StatusCode)
	}
	return nil
}

func (c *CouchDB) ensureDatabase(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+"/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != 412 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gagal membuat database %s: status %d, body %s", name, resp.StatusCode, string(body))
	}
	log.Printf("Database %s siap", name)
	return nil
}

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
	if err := c.getDoc(ctx, "alumni", id, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *CouchDB) SaveAlumni(ctx context.Context, a *models.Alumni) error {
	id, rev, err := c.postDoc(ctx, "alumni", a)
	if err != nil {
		return err
	}
	a.ID = id
	a.Rev = rev
	return nil
}

func (c *CouchDB) UpdateAlumni(ctx context.Context, a *models.Alumni) error {
	rev, err := c.putDoc(ctx, "alumni", a.ID, a.Rev, a)
	if err != nil {
		return err
	}
	a.Rev = rev
	return nil
}

func (c *CouchDB) ListAlumni(ctx context.Context) ([]models.Alumni, error) {
	var result struct {
		Rows []struct {
			Doc models.Alumni `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "alumni", &result); err != nil {
		return nil, err
	}
	var alumni []models.Alumni
	for _, row := range result.Rows {
		alumni = append(alumni, row.Doc)
	}
	return alumni, nil
}

func (c *CouchDB) FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error) {
	var result struct {
		Docs []models.Alumni `json:"docs"`
	}
	if err := c.findDocs(ctx, "alumni", map[string]interface{}{"email": email}, &result); err != nil {
		return nil, err
	}
	if len(result.Docs) == 0 {
		return nil, nil
	}
	return &result.Docs[0], nil
}

func (c *CouchDB) FindAlumniByVerificationToken(ctx context.Context, token string) (*models.Alumni, error) {
	var result struct {
		Docs []models.Alumni `json:"docs"`
	}
	if err := c.findDocs(ctx, "alumni", map[string]interface{}{"verification_token": token}, &result); err != nil {
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
	if err := c.getDoc(ctx, "subgroups", id, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *CouchDB) SaveSubgroup(ctx context.Context, s *models.Subgroup) error {
	id, rev, err := c.postDoc(ctx, "subgroups", s)
	if err != nil {
		return err
	}
	s.ID = id
	s.Rev = rev
	return nil
}

func (c *CouchDB) UpdateSubgroup(ctx context.Context, s *models.Subgroup) error {
	rev, err := c.putDoc(ctx, "subgroups", s.ID, s.Rev, s)
	if err != nil {
		return err
	}
	s.Rev = rev
	return nil
}

func (c *CouchDB) DeleteSubgroup(ctx context.Context, id, rev string) error {
	return c.deleteDoc(ctx, "subgroups", id, rev)
}

func (c *CouchDB) ListSubgroups(ctx context.Context) ([]models.Subgroup, error) {
	var result struct {
		Rows []struct {
			Doc models.Subgroup `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "subgroups", &result); err != nil {
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
	id, rev, err := c.postDoc(ctx, "forum_posts", p)
	if err != nil {
		return err
	}
	p.ID = id
	p.Rev = rev
	return nil
}

func (c *CouchDB) ListForumPosts(ctx context.Context) ([]models.ForumPost, error) {
	var result struct {
		Rows []struct {
			Doc models.ForumPost `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "forum_posts", &result, "descending=true"); err != nil {
		return nil, err
	}
	var posts []models.ForumPost
	for _, row := range result.Rows {
		posts = append(posts, row.Doc)
	}
	return posts, nil
}

// --- internal helpers ---

func (c *CouchDB) getDoc(ctx context.Context, dbName, id string, into interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/"+dbName+"/"+id, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("get doc failed: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func (c *CouchDB) postDoc(ctx context.Context, dbName string, doc interface{}) (id, rev string, err error) {
	body, err := json.Marshal(doc)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/"+dbName, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("post doc failed: status %d", resp.StatusCode)
	}
	var result struct {
		ID  string `json:"id"`
		Rev string `json:"rev"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	return result.ID, result.Rev, nil
}

func (c *CouchDB) putDoc(ctx context.Context, dbName, id, rev string, doc interface{}) (string, error) {
	body, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+"/"+dbName+"/"+id+"?rev="+rev, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("put doc failed: status %d", resp.StatusCode)
	}
	var result struct {
		Rev string `json:"rev"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Rev, nil
}

func (c *CouchDB) deleteDoc(ctx context.Context, dbName, id, rev string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/"+dbName+"/"+id+"?rev="+rev, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete doc failed: status %d", resp.StatusCode)
	}
	return nil
}

func (c *CouchDB) allDocs(ctx context.Context, dbName string, into interface{}, opts ...string) error {
	url := c.baseURL + "/" + dbName + "/_all_docs?include_docs=true"
	for _, o := range opts {
		url += "&" + o
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("all_docs failed: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func (c *CouchDB) findDocs(ctx context.Context, dbName string, selector interface{}, into interface{}) error {
	body, err := json.Marshal(map[string]interface{}{"selector": selector})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/"+dbName+"/_find", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("find failed: status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

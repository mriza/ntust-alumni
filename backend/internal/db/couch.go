package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"ntust-alumni/backend/internal/models"
)

// Store adalah interface untuk operasi database alumni, subgroup, dan forum post.
type Store interface {
	GetAlumni(ctx context.Context, id string) (*models.Alumni, error)
	SaveAlumni(ctx context.Context, a *models.Alumni) error
	UpdateAlumni(ctx context.Context, a *models.Alumni) error
	ListAlumni(ctx context.Context, search string, wilayah string, jurusan string, limit int, offset int) ([]models.Alumni, error)
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
	baseURL  string
	username string
	password string
	client   *http.Client
}

func NewCouchDB(rawURL string) (*CouchDB, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, errors.New("gagal parsing url couchdb")
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		u.User = nil
	}

	cleanURL := strings.TrimRight(u.String(), "/")
	c := &CouchDB{
		baseURL:  cleanURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	if err := c.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke couchdb: %w", err)
	}

	log.Println("Terhubung ke CouchDB")
	return c, nil
}

func (c *CouchDB) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return req, nil
}

func validateDocID(id string) error {
	if id == "" || strings.HasPrefix(id, "_") || strings.Contains(id, "..") {
		return errors.New("id dokumen tidak valid")
	}
	for _, r := range id {
		if r == '/' || r == '\\' || r == '?' || r == '#' || unicode.IsControl(r) || unicode.IsSpace(r) {
			return errors.New("id dokumen mengandung karakter tidak valid")
		}
	}
	return nil
}

func (c *CouchDB) Ping() error {
	req, err := c.newRequest(context.Background(), http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
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
	req, err := c.newRequest(ctx, http.MethodPut, "/"+url.PathEscape(name), nil)
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

// --- CouchDB Wire Documents ---

type couchAlumniDoc struct {
	ID               string                  `json:"_id,omitempty"`
	Rev              string                  `json:"_rev,omitempty"`
	Nama             string                  `json:"nama"`
	Email            string                  `json:"email"`
	PasswordHash     string                  `json:"password_hash,omitempty"`
	PasswordLegacy   string                  `json:"password,omitempty"`
	Nim              string                  `json:"nim"`
	Jurusan          string                  `json:"jurusan,omitempty"`
	Status           string                  `json:"status"`
	EmailNtust       string                  `json:"email_ntust,omitempty"`
	NoTelp           string                  `json:"no_telp,omitempty"`
	NoWA             string                  `json:"no_wa,omitempty"`
	Telegram         string                  `json:"telegram,omitempty"`
	Sosmed           *models.Sosmed          `json:"sosmed,omitempty"`
	FotoProfil       string                  `json:"foto_profil,omitempty"`
	LinkedIn         string                  `json:"linkedin,omitempty"`
	BidangKeahianan  []string                `json:"bidang_keahlian,omitempty"`
	PreferensiKontak string                  `json:"preferensi_kontak,omitempty"`
	StatusKeaktifan  string                  `json:"status_keaktifan,omitempty"`
	Pekerjaan        *models.Pekerjaan       `json:"pekerjaan,omitempty"`
	PendidikanNTUST  *models.PendidikanNTUST `json:"pendidikan_ntust,omitempty"`
	Wilayah          string                  `json:"wilayah,omitempty"`
	SubgroupID       string                  `json:"subgroup_id,omitempty"`
	Role             string                  `json:"role"`
	OtpHash             string    `json:"otp_hash,omitempty"`
	OtpExpiresAt        time.Time `json:"otp_expires_at,omitempty"`
	VerificationToken   string    `json:"verification_token,omitempty"`
	TokenExpiresAt      time.Time `json:"token_expires_at,omitempty"`
	VerificationChannel string    `json:"verification_channel,omitempty"`
	VerificationTarget  string    `json:"verification_target,omitempty"`
	DibuatPada          time.Time `json:"dibuat_pada,omitempty"`
	Diperbarui       time.Time               `json:"diperbarui,omitempty"`
}

func toCouchAlumniDoc(a *models.Alumni) couchAlumniDoc {
	return couchAlumniDoc{
		ID:                a.ID,
		Rev:               a.Rev,
		Nama:              a.Nama,
		Email:             a.Email,
		PasswordHash:      a.PasswordHash,
		Nim:               a.Nim,
		Jurusan:           a.Jurusan,
		Status:            a.Status,
		EmailNtust:        a.EmailNtust,
		NoTelp:            a.NoTelp,
		NoWA:              a.NoWA,
		Telegram:          a.Telegram,
		Sosmed:            a.Sosmed,
		FotoProfil:        a.FotoProfil,
		LinkedIn:          a.LinkedIn,
		BidangKeahianan:   a.BidangKeahianan,
		PreferensiKontak:  a.PreferensiKontak,
		StatusKeaktifan:   a.StatusKeaktifan,
		Pekerjaan:         a.Pekerjaan,
		PendidikanNTUST:   a.PendidikanNTUST,
		Wilayah:           a.Wilayah,
		SubgroupID:        a.SubgroupID,
		Role:              a.Role,
		OtpHash:             a.OtpHash,
		OtpExpiresAt:        a.OtpExpiresAt,
		VerificationToken:   a.VerificationToken,
		TokenExpiresAt:      a.TokenExpiresAt,
		VerificationChannel: a.VerificationChannel,
		VerificationTarget:  a.VerificationTarget,
		DibuatPada:          a.DibuatPada,
		Diperbarui:        a.Diperbarui,
	}
}

func fromCouchAlumniDoc(doc couchAlumniDoc) models.Alumni {
	pwHash := doc.PasswordHash
	if pwHash == "" && doc.PasswordLegacy != "" {
		pwHash = doc.PasswordLegacy
	}
	return models.Alumni{
		ID:                doc.ID,
		Rev:               doc.Rev,
		Nama:              doc.Nama,
		Email:             doc.Email,
		PasswordHash:      pwHash,
		Nim:               doc.Nim,
		Jurusan:           doc.Jurusan,
		Status:            doc.Status,
		EmailNtust:        doc.EmailNtust,
		NoTelp:            doc.NoTelp,
		NoWA:              doc.NoWA,
		Telegram:          doc.Telegram,
		Sosmed:            doc.Sosmed,
		FotoProfil:        doc.FotoProfil,
		LinkedIn:          doc.LinkedIn,
		BidangKeahianan:   doc.BidangKeahianan,
		PreferensiKontak:  doc.PreferensiKontak,
		StatusKeaktifan:   doc.StatusKeaktifan,
		Pekerjaan:         doc.Pekerjaan,
		PendidikanNTUST:   doc.PendidikanNTUST,
		Wilayah:           doc.Wilayah,
		SubgroupID:        doc.SubgroupID,
		Role:              doc.Role,
		OtpHash:             doc.OtpHash,
		OtpExpiresAt:        doc.OtpExpiresAt,
		VerificationToken:   doc.VerificationToken,
		TokenExpiresAt:      doc.TokenExpiresAt,
		VerificationChannel: doc.VerificationChannel,
		VerificationTarget:  doc.VerificationTarget,
		DibuatPada:          doc.DibuatPada,
		Diperbarui:        doc.Diperbarui,
	}
}

type couchSubgroupDoc struct {
	ID         string                   `json:"_id,omitempty"`
	Rev        string                   `json:"_rev,omitempty"`
	Nama       string                   `json:"nama"`
	Wilayah    string                   `json:"wilayah"`
	Deskripsi  string                   `json:"deskripsi"`
	AdminID    string                   `json:"admin_id"`
	Pengurus   []models.PengurusWilayah `json:"pengurus,omitempty"`
	DibuatPada time.Time                `json:"dibuat_pada,omitempty"`
	Diperbarui time.Time                `json:"diperbarui,omitempty"`
}

func toCouchSubgroupDoc(s *models.Subgroup) couchSubgroupDoc {
	return couchSubgroupDoc{
		ID:         s.ID,
		Rev:        s.Rev,
		Nama:       s.Nama,
		Wilayah:    s.Wilayah,
		Deskripsi:  s.Deskripsi,
		AdminID:    s.AdminID,
		Pengurus:   s.Pengurus,
		DibuatPada: s.DibuatPada,
		Diperbarui: s.Diperbarui,
	}
}

func fromCouchSubgroupDoc(doc couchSubgroupDoc) models.Subgroup {
	return models.Subgroup{
		ID:         doc.ID,
		Rev:        doc.Rev,
		Nama:       doc.Nama,
		Wilayah:    doc.Wilayah,
		Deskripsi:  doc.Deskripsi,
		AdminID:    doc.AdminID,
		Pengurus:   doc.Pengurus,
		DibuatPada: doc.DibuatPada,
		Diperbarui: doc.Diperbarui,
	}
}

type couchForumPostDoc struct {
	ID          string    `json:"_id,omitempty"`
	Rev         string    `json:"_rev,omitempty"`
	Judul       string    `json:"judul"`
	Isi         string    `json:"isi"`
	PenulisID   string    `json:"penulis_id"`
	PenulisNama string    `json:"penulis_nama"`
	Wilayah     string    `json:"wilayah,omitempty"`
	DibuatPada  time.Time `json:"dibuat_pada,omitempty"`
	Diperbarui  time.Time `json:"diperbarui,omitempty"`
}

func toCouchForumPostDoc(p *models.ForumPost) couchForumPostDoc {
	return couchForumPostDoc{
		ID:          p.ID,
		Rev:         p.Rev,
		Judul:       p.Judul,
		Isi:         p.Isi,
		PenulisID:   p.PenulisID,
		PenulisNama: p.PenulisNama,
		Wilayah:     p.Wilayah,
		DibuatPada:  p.DibuatPada,
		Diperbarui:  p.Diperbarui,
	}
}

func fromCouchForumPostDoc(doc couchForumPostDoc) models.ForumPost {
	return models.ForumPost{
		ID:          doc.ID,
		Rev:         doc.Rev,
		Judul:       doc.Judul,
		Isi:         doc.Isi,
		PenulisID:   doc.PenulisID,
		PenulisNama: doc.PenulisNama,
		Wilayah:     doc.Wilayah,
		DibuatPada:  doc.DibuatPada,
		Diperbarui:  doc.Diperbarui,
	}
}

// --- Alumni ---

func (c *CouchDB) GetAlumni(ctx context.Context, id string) (*models.Alumni, error) {
	var doc couchAlumniDoc
	if err := c.getDoc(ctx, "alumni", id, &doc); err != nil {
		return nil, err
	}
	a := fromCouchAlumniDoc(doc)
	return &a, nil
}

func (c *CouchDB) SaveAlumni(ctx context.Context, a *models.Alumni) error {
	now := time.Now().UTC()
	if a.DibuatPada.IsZero() {
		a.DibuatPada = now
	}
	a.Diperbarui = now
	id, rev, err := c.postDoc(ctx, "alumni", toCouchAlumniDoc(a))
	if err != nil {
		return err
	}
	a.ID = id
	a.Rev = rev
	return nil
}

func (c *CouchDB) UpdateAlumni(ctx context.Context, a *models.Alumni) error {
	a.Diperbarui = time.Now().UTC()
	rev, err := c.putDoc(ctx, "alumni", a.ID, a.Rev, toCouchAlumniDoc(a))
	if err != nil {
		return err
	}
	a.Rev = rev
	return nil
}

func (c *CouchDB) ListAlumni(ctx context.Context, search, wilayah, jurusan string, limit, offset int) ([]models.Alumni, error) {
	// Build selector for CouchDB _find with filters
	selector := map[string]interface{}{}
	if wilayah != "" {
		selector["wilayah"] = wilayah
	}
	if jurusan != "" {
		selector["jurusan"] = jurusan
	}
	if len(selector) > 0 {
		var result struct {
			Docs []couchAlumniDoc `json:"docs"`
		}
		if err := c.findDocs(ctx, "alumni", selector, &result); err != nil {
			return nil, err
		}
		var alumni []models.Alumni
		for _, doc := range result.Docs {
			a := fromCouchAlumniDoc(doc)
			if applyAlumniSearch(&a, search) {
				alumni = append(alumni, a)
			}
		}
		return paginateAlumni(alumni, limit, offset), nil
	}

	// Fallback: scan all when no selector filters
	var result struct {
		Rows []struct {
			Doc couchAlumniDoc `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "alumni", &result); err != nil {
		return nil, err
	}
	var alumni []models.Alumni
	for _, row := range result.Rows {
		a := fromCouchAlumniDoc(row.Doc)
		if applyAlumniSearch(&a, search) {
			alumni = append(alumni, a)
		}
	}
	return paginateAlumni(alumni, limit, offset), nil
}

func applyAlumniSearch(a *models.Alumni, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(a.Nama), q) ||
		strings.Contains(strings.ToLower(a.Email), q) ||
		strings.Contains(strings.ToLower(a.Nim), q) ||
		strings.Contains(strings.ToLower(a.Wilayah), q) ||
		strings.Contains(strings.ToLower(a.Jurusan), q) {
		return true
	}
	if a.Pekerjaan != nil && (strings.Contains(strings.ToLower(a.Pekerjaan.Institusi), q) ||
		strings.Contains(strings.ToLower(a.Pekerjaan.Jabatan), q)) {
		return true
	}
	if a.PendidikanNTUST != nil && (strings.Contains(strings.ToLower(a.PendidikanNTUST.Fakultas), q) ||
		strings.Contains(strings.ToLower(a.PendidikanNTUST.Jurusan), q)) {
		return true
	}
	return false
}

func paginateAlumni(list []models.Alumni, limit, offset int) []models.Alumni {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(list) {
		return []models.Alumni{}
	}
	if limit <= 0 {
		return list[offset:]
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end]
}

func (c *CouchDB) FindAlumniByEmail(ctx context.Context, email string) (*models.Alumni, error) {
	var result struct {
		Docs []couchAlumniDoc `json:"docs"`
	}
	if err := c.findDocs(ctx, "alumni", map[string]interface{}{"email": email}, &result); err != nil {
		return nil, err
	}
	if len(result.Docs) == 0 {
		return nil, nil
	}
	a := fromCouchAlumniDoc(result.Docs[0])
	return &a, nil
}

func (c *CouchDB) FindAlumniByVerificationToken(ctx context.Context, token string) (*models.Alumni, error) {
	var result struct {
		Docs []couchAlumniDoc `json:"docs"`
	}
	if err := c.findDocs(ctx, "alumni", map[string]interface{}{"verification_token": token}, &result); err != nil {
		return nil, err
	}
	if len(result.Docs) == 0 {
		return nil, nil
	}
	a := fromCouchAlumniDoc(result.Docs[0])
	return &a, nil
}

// --- Subgroup ---

func (c *CouchDB) GetSubgroup(ctx context.Context, id string) (*models.Subgroup, error) {
	var doc couchSubgroupDoc
	if err := c.getDoc(ctx, "subgroups", id, &doc); err != nil {
		return nil, err
	}
	s := fromCouchSubgroupDoc(doc)
	return &s, nil
}

func (c *CouchDB) SaveSubgroup(ctx context.Context, s *models.Subgroup) error {
	now := time.Now().UTC()
	if s.DibuatPada.IsZero() {
		s.DibuatPada = now
	}
	s.Diperbarui = now
	id, rev, err := c.postDoc(ctx, "subgroups", toCouchSubgroupDoc(s))
	if err != nil {
		return err
	}
	s.ID = id
	s.Rev = rev
	return nil
}

func (c *CouchDB) UpdateSubgroup(ctx context.Context, s *models.Subgroup) error {
	s.Diperbarui = time.Now().UTC()
	rev, err := c.putDoc(ctx, "subgroups", s.ID, s.Rev, toCouchSubgroupDoc(s))
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
			Doc couchSubgroupDoc `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "subgroups", &result); err != nil {
		return nil, err
	}
	subgroups := make([]models.Subgroup, 0, len(result.Rows))
	for _, row := range result.Rows {
		subgroups = append(subgroups, fromCouchSubgroupDoc(row.Doc))
	}
	return subgroups, nil
}

// --- Forum Post ---

func (c *CouchDB) SaveForumPost(ctx context.Context, p *models.ForumPost) error {
	now := time.Now().UTC()
	if p.DibuatPada.IsZero() {
		p.DibuatPada = now
	}
	p.Diperbarui = now
	id, rev, err := c.postDoc(ctx, "forum_posts", toCouchForumPostDoc(p))
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
			Doc couchForumPostDoc `json:"doc"`
		} `json:"rows"`
	}
	if err := c.allDocs(ctx, "forum_posts", &result, "descending=true"); err != nil {
		return nil, err
	}
	posts := make([]models.ForumPost, 0, len(result.Rows))
	for _, row := range result.Rows {
		posts = append(posts, fromCouchForumPostDoc(row.Doc))
	}
	return posts, nil
}

// --- internal helpers ---

func (c *CouchDB) getDoc(ctx context.Context, dbName, id string, into interface{}) error {
	if err := validateDocID(id); err != nil {
		return fmt.Errorf("not found")
	}
	req, err := c.newRequest(ctx, http.MethodGet, "/"+url.PathEscape(dbName)+"/"+url.PathEscape(id), nil)
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
	req, err := c.newRequest(ctx, http.MethodPost, "/"+url.PathEscape(dbName), bytes.NewReader(body))
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
	if err := validateDocID(id); err != nil {
		return "", err
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	req, err := c.newRequest(ctx, http.MethodPut, "/"+url.PathEscape(dbName)+"/"+url.PathEscape(id)+"?rev="+url.QueryEscape(rev), bytes.NewReader(body))
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
	if err := validateDocID(id); err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodDelete, "/"+url.PathEscape(dbName)+"/"+url.PathEscape(id)+"?rev="+url.QueryEscape(rev), nil)
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
	path := "/" + url.PathEscape(dbName) + "/_all_docs?include_docs=true"
	for _, o := range opts {
		path += "&" + o
	}
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
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
	body, err := json.Marshal(map[string]interface{}{"selector": selector, "limit": 1000})
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/"+url.PathEscape(dbName)+"/_find", bytes.NewReader(body))
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

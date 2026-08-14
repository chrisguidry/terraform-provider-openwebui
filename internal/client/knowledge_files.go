package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// KnowledgeFileIDForm associates a file with a knowledge base.
type KnowledgeFileIDForm struct {
	FileID string `json:"file_id"`
}

// knowledgeFilesPageSize mirrors the backend PAGE_ITEM_COUNT for
// GET /knowledge/{id}/files.
const knowledgeFilesPageSize = 30

// KnowledgeFileListResponse captures paginated file results for a knowledge base.
type KnowledgeFileListResponse struct {
	Items []FileUserResponse `json:"items"`
	Total int                `json:"total"`
}

// FileUserResponse captures file details including user metadata. The knowledge
// file listing defers the file's extracted content, so this record holds
// metadata only.
type FileUserResponse struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Hash      *string  `json:"hash,omitempty"`
	Filename  string   `json:"filename"`
	Meta      FileMeta `json:"meta"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
	User      *User    `json:"user,omitempty"`
}

// getKnowledgeFilePage reads one page of a knowledge base's file attachments.
func (c *Client) getKnowledgeFilePage(ctx context.Context, knowledgeID string, page int) (*KnowledgeFileListResponse, error) {
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", page))

	var resp KnowledgeFileListResponse
	path := fmt.Sprintf("knowledge/%s/files", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListKnowledgeFiles retrieves every file attached to a knowledge base, paging
// until the server runs out of items. The route caps a page at 30 files, so a
// single request sees only part of a large knowledge base.
func (c *Client) ListKnowledgeFiles(ctx context.Context, knowledgeID string) ([]FileUserResponse, error) {
	var all []FileUserResponse
	for page := 1; ; page++ {
		resp, err := c.getKnowledgeFilePage(ctx, knowledgeID, page)
		if err != nil {
			return nil, err
		}

		all = append(all, resp.Items...)
		if len(resp.Items) < knowledgeFilesPageSize || len(all) >= resp.Total {
			break
		}
	}

	return all, nil
}

// CountKnowledgeFiles returns the number of files attached to a knowledge base.
// The count comes from the listing's total, which the server computes before it
// paginates.
func (c *Client) CountKnowledgeFiles(ctx context.Context, knowledgeID string) (int, error) {
	resp, err := c.getKnowledgeFilePage(ctx, knowledgeID, 1)
	if err != nil {
		return 0, err
	}

	return resp.Total, nil
}

// AddKnowledgeFile associates a file with a knowledge base.
func (c *Client) AddKnowledgeFile(ctx context.Context, knowledgeID string, fileID string) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	path := fmt.Sprintf("knowledge/%s/file/add", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateKnowledgeFile updates a file association within a knowledge base.
func (c *Client) UpdateKnowledgeFile(ctx context.Context, knowledgeID string, fileID string) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	path := fmt.Sprintf("knowledge/%s/file/update", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveKnowledgeFile detaches a file from a knowledge base.
func (c *Client) RemoveKnowledgeFile(ctx context.Context, knowledgeID string, fileID string, deleteFile bool) (*KnowledgeFilesResponse, error) {
	var resp KnowledgeFilesResponse
	form := KnowledgeFileIDForm{FileID: fileID}
	query := url.Values{"delete_file": []string{fmt.Sprintf("%t", deleteFile)}}
	path := fmt.Sprintf("knowledge/%s/file/remove", url.PathEscape(knowledgeID))
	if err := c.do(ctx, http.MethodPost, path, query, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

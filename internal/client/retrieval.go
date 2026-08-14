package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// NumericInt carries a config value the form declares as an integer but whose
// stored default is a string, so a read has to accept both. rag.mineru_api_timeout
// is the one such key in the retrieval config.
type NumericInt int64

// UnmarshalJSON accepts a number or a string holding a number.
func (n *NumericInt) UnmarshalJSON(data []byte) error {
	var number int64
	if err := json.Unmarshal(data, &number); err == nil {
		*n = NumericInt(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("expected a number or a string holding a number, got %s", data)
	}

	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("expected a number or a string holding a number, got %s", data)
	}

	*n = NumericInt(parsed)

	return nil
}

// NumericString carries the four rag FILE_* keys. Open WebUI stores them as
// integers, reads them back as either integers or strings, and treats the empty
// string as the value that clears the limit
// (backend/open_webui/routers/retrieval.py, lines 1216 to 1227).
type NumericString string

// UnmarshalJSON accepts a number or a string.
func (n *NumericString) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*n = NumericString(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("expected a string or a number, got %s", data)
	}

	*n = NumericString(number.String())

	return nil
}

// MarshalJSON sends a whole number as a number, so the stored value keeps the
// type the rest of Open WebUI reads it as, and sends the empty string as it
// stands because that is what clears the limit.
func (n NumericString) MarshalJSON() ([]byte, error) {
	if n == "" {
		return []byte(`""`), nil
	}

	if _, err := strconv.ParseInt(string(n), 10, 64); err == nil {
		return []byte(n), nil
	}

	return json.Marshal(string(n))
}

// RAGEngineConfigForm carries the url and key of one embedding engine.
type RAGEngineConfigForm struct {
	URL     *string `json:"url"`
	Key     *string `json:"key"`
	Version *string `json:"version,omitempty"`
}

// RAGEmbeddingConfigForm mirrors EmbeddingModelUpdateForm in
// backend/open_webui/routers/retrieval.py. The update route writes an engine
// block only when RAG_EMBEDDING_ENGINE is ollama, openai, or azure_openai, and
// a block left out keeps its stored values.
type RAGEmbeddingConfigForm struct {
	RAGEmbeddingEngine             *string              `json:"RAG_EMBEDDING_ENGINE,omitempty"`
	RAGEmbeddingModel              *string              `json:"RAG_EMBEDDING_MODEL,omitempty"`
	RAGEmbeddingBatchSize          *int64               `json:"RAG_EMBEDDING_BATCH_SIZE,omitempty"`
	EnableAsyncEmbedding           *bool                `json:"ENABLE_ASYNC_EMBEDDING,omitempty"`
	RAGEmbeddingConcurrentRequests *int64               `json:"RAG_EMBEDDING_CONCURRENT_REQUESTS,omitempty"`
	OpenAIConfig                   *RAGEngineConfigForm `json:"openai_config,omitempty"`
	OllamaConfig                   *RAGEngineConfigForm `json:"ollama_config,omitempty"`
	AzureOpenAIConfig              *RAGEngineConfigForm `json:"azure_openai_config,omitempty"`
}

// RAGConfigForm mirrors ConfigForm in
// backend/open_webui/routers/retrieval.py. Every top-level key keeps its stored
// value when the request omits it.
type RAGConfigForm struct {
	RAGTemplate                              *string           `json:"RAG_TEMPLATE,omitempty"`
	TopK                                     *int64            `json:"TOP_K,omitempty"`
	BypassEmbeddingAndRetrieval              *bool             `json:"BYPASS_EMBEDDING_AND_RETRIEVAL,omitempty"`
	RAGFullContext                           *bool             `json:"RAG_FULL_CONTEXT,omitempty"`
	EnableRAGHybridSearch                    *bool             `json:"ENABLE_RAG_HYBRID_SEARCH,omitempty"`
	EnableRAGHybridSearchEnrichedTexts       *bool             `json:"ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS,omitempty"`
	TopKReranker                             *int64            `json:"TOP_K_RERANKER,omitempty"`
	RelevanceThreshold                       *float64          `json:"RELEVANCE_THRESHOLD,omitempty"`
	HybridBM25Weight                         *float64          `json:"HYBRID_BM25_WEIGHT,omitempty"`
	ContentExtractionEngine                  *string           `json:"CONTENT_EXTRACTION_ENGINE,omitempty"`
	ContentExtractionSupportedMediaMimeTypes *[]string         `json:"CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES,omitempty"`
	PDFExtractImages                         *bool             `json:"PDF_EXTRACT_IMAGES,omitempty"`
	PDFLoaderMode                            *string           `json:"PDF_LOADER_MODE,omitempty"`
	DatalabMarkerAPIKey                      *string           `json:"DATALAB_MARKER_API_KEY,omitempty"`
	DatalabMarkerAPIBaseURL                  *string           `json:"DATALAB_MARKER_API_BASE_URL,omitempty"`
	DatalabMarkerAdditionalConfig            *string           `json:"DATALAB_MARKER_ADDITIONAL_CONFIG,omitempty"`
	DatalabMarkerSkipCache                   *bool             `json:"DATALAB_MARKER_SKIP_CACHE,omitempty"`
	DatalabMarkerForceOCR                    *bool             `json:"DATALAB_MARKER_FORCE_OCR,omitempty"`
	DatalabMarkerPaginate                    *bool             `json:"DATALAB_MARKER_PAGINATE,omitempty"`
	DatalabMarkerStripExistingOCR            *bool             `json:"DATALAB_MARKER_STRIP_EXISTING_OCR,omitempty"`
	DatalabMarkerDisableImageExtraction      *bool             `json:"DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION,omitempty"`
	DatalabMarkerFormatLines                 *bool             `json:"DATALAB_MARKER_FORMAT_LINES,omitempty"`
	DatalabMarkerUseLLM                      *bool             `json:"DATALAB_MARKER_USE_LLM,omitempty"`
	DatalabMarkerOutputFormat                *string           `json:"DATALAB_MARKER_OUTPUT_FORMAT,omitempty"`
	ExternalDocumentLoaderURL                *string           `json:"EXTERNAL_DOCUMENT_LOADER_URL,omitempty"`
	ExternalDocumentLoaderAPIKey             *string           `json:"EXTERNAL_DOCUMENT_LOADER_API_KEY,omitempty"`
	ExternalDocumentLoaderHeaders            any               `json:"EXTERNAL_DOCUMENT_LOADER_HEADERS,omitempty"`
	TikaServerURL                            *string           `json:"TIKA_SERVER_URL,omitempty"`
	DoclingServerURL                         *string           `json:"DOCLING_SERVER_URL,omitempty"`
	DoclingAPIKey                            *string           `json:"DOCLING_API_KEY,omitempty"`
	DoclingParams                            any               `json:"DOCLING_PARAMS,omitempty"`
	DocumentIntelligenceEndpoint             *string           `json:"DOCUMENT_INTELLIGENCE_ENDPOINT,omitempty"`
	DocumentIntelligenceKey                  *string           `json:"DOCUMENT_INTELLIGENCE_KEY,omitempty"`
	DocumentIntelligenceModel                *string           `json:"DOCUMENT_INTELLIGENCE_MODEL,omitempty"`
	MistralOCRAPIBaseURL                     *string           `json:"MISTRAL_OCR_API_BASE_URL,omitempty"`
	MistralOCRAPIKey                         *string           `json:"MISTRAL_OCR_API_KEY,omitempty"`
	MistralOCRUseBase64                      *bool             `json:"MISTRAL_OCR_USE_BASE64,omitempty"`
	PaddleocrVLBaseURL                       *string           `json:"PADDLEOCR_VL_BASE_URL,omitempty"`
	PaddleocrVLToken                         *string           `json:"PADDLEOCR_VL_TOKEN,omitempty"`
	MineruAPIMode                            *string           `json:"MINERU_API_MODE,omitempty"`
	MineruAPIURL                             *string           `json:"MINERU_API_URL,omitempty"`
	MineruAPIKey                             *string           `json:"MINERU_API_KEY,omitempty"`
	MineruAPITimeout                         *NumericInt       `json:"MINERU_API_TIMEOUT,omitempty"`
	MineruParams                             any               `json:"MINERU_PARAMS,omitempty"`
	MineruFileExtensions                     *[]string         `json:"MINERU_FILE_EXTENSIONS,omitempty"`
	RAGRerankingModel                        *string           `json:"RAG_RERANKING_MODEL,omitempty"`
	RAGRerankingEngine                       *string           `json:"RAG_RERANKING_ENGINE,omitempty"`
	RAGRerankingBatchSize                    *int64            `json:"RAG_RERANKING_BATCH_SIZE,omitempty"`
	RAGExternalRerankerURL                   *string           `json:"RAG_EXTERNAL_RERANKER_URL,omitempty"`
	RAGExternalRerankerAPIKey                *string           `json:"RAG_EXTERNAL_RERANKER_API_KEY,omitempty"`
	RAGExternalRerankerTimeout               *string           `json:"RAG_EXTERNAL_RERANKER_TIMEOUT,omitempty"`
	TextSplitter                             *string           `json:"TEXT_SPLITTER,omitempty"`
	RAGTokenizerModel                        *string           `json:"RAG_TOKENIZER_MODEL,omitempty"`
	EnableMarkdownHeaderTextSplitter         *bool             `json:"ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER,omitempty"`
	ChunkSize                                *int64            `json:"CHUNK_SIZE,omitempty"`
	ChunkMinSizeTarget                       *int64            `json:"CHUNK_MIN_SIZE_TARGET,omitempty"`
	ChunkOverlap                             *int64            `json:"CHUNK_OVERLAP,omitempty"`
	FileMaxSize                              *NumericString    `json:"FILE_MAX_SIZE,omitempty"`
	FileMaxCount                             *NumericString    `json:"FILE_MAX_COUNT,omitempty"`
	FileImageCompressionWidth                *NumericString    `json:"FILE_IMAGE_COMPRESSION_WIDTH,omitempty"`
	FileImageCompressionHeight               *NumericString    `json:"FILE_IMAGE_COMPRESSION_HEIGHT,omitempty"`
	AllowedFileExtensions                    *[]string         `json:"ALLOWED_FILE_EXTENSIONS,omitempty"`
	EnableGoogleDriveIntegration             *bool             `json:"ENABLE_GOOGLE_DRIVE_INTEGRATION,omitempty"`
	EnableOneDriveIntegration                *bool             `json:"ENABLE_ONEDRIVE_INTEGRATION,omitempty"`
	Web                                      *RAGWebConfigForm `json:"web,omitempty"`
}

// RAGWebConfigForm mirrors WebConfig in
// backend/open_webui/routers/retrieval.py, less YOUTUBE_LOADER_TRANSLATION, which
// the update route keeps in process memory instead of storage. Once the request
// carries a web block the route assigns every key of it, so an omitted key is
// written as its Pydantic default.
type RAGWebConfigForm struct {
	EnableWebSearch                      *bool     `json:"ENABLE_WEB_SEARCH,omitempty"`
	EnableWebSearchConfirmation          *bool     `json:"ENABLE_WEB_SEARCH_CONFIRMATION,omitempty"`
	WebSearchConfirmationContent         *string   `json:"WEB_SEARCH_CONFIRMATION_CONTENT,omitempty"`
	WebSearchEngine                      *string   `json:"WEB_SEARCH_ENGINE,omitempty"`
	WebSearchTrustEnv                    *bool     `json:"WEB_SEARCH_TRUST_ENV,omitempty"`
	WebSearchResultCount                 *int64    `json:"WEB_SEARCH_RESULT_COUNT,omitempty"`
	WebSearchConcurrentRequests          *int64    `json:"WEB_SEARCH_CONCURRENT_REQUESTS,omitempty"`
	WebSearchDomainFilterList            *[]string `json:"WEB_SEARCH_DOMAIN_FILTER_LIST,omitempty"`
	WebFetchMaxContentLength             *int64    `json:"WEB_FETCH_MAX_CONTENT_LENGTH,omitempty"`
	WebLoaderConcurrentRequests          *int64    `json:"WEB_LOADER_CONCURRENT_REQUESTS,omitempty"`
	BypassWebSearchEmbeddingAndRetrieval *bool     `json:"BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL,omitempty"`
	BypassWebSearchWebLoader             *bool     `json:"BYPASS_WEB_SEARCH_WEB_LOADER,omitempty"`
	OllamaCloudWebSearchAPIKey           *string   `json:"OLLAMA_CLOUD_WEB_SEARCH_API_KEY,omitempty"`
	SearxngQueryURL                      *string   `json:"SEARXNG_QUERY_URL,omitempty"`
	SearxngLanguage                      *string   `json:"SEARXNG_LANGUAGE,omitempty"`
	OpenserpBaseURL                      *string   `json:"OPENSERP_BASE_URL,omitempty"`
	YacyQueryURL                         *string   `json:"YACY_QUERY_URL,omitempty"`
	YacyUsername                         *string   `json:"YACY_USERNAME,omitempty"`
	YacyPassword                         *string   `json:"YACY_PASSWORD,omitempty"`
	GooglePSEAPIKey                      *string   `json:"GOOGLE_PSE_API_KEY,omitempty"`
	GooglePSEEngineID                    *string   `json:"GOOGLE_PSE_ENGINE_ID,omitempty"`
	BraveSearchAPIKey                    *string   `json:"BRAVE_SEARCH_API_KEY,omitempty"`
	BraveSearchContextTokens             *int64    `json:"BRAVE_SEARCH_CONTEXT_TOKENS,omitempty"`
	KagiSearchAPIKey                     *string   `json:"KAGI_SEARCH_API_KEY,omitempty"`
	MojeekSearchAPIKey                   *string   `json:"MOJEEK_SEARCH_API_KEY,omitempty"`
	BochaSearchAPIKey                    *string   `json:"BOCHA_SEARCH_API_KEY,omitempty"`
	SerpstackAPIKey                      *string   `json:"SERPSTACK_API_KEY,omitempty"`
	SerpstackHTTPS                       *bool     `json:"SERPSTACK_HTTPS,omitempty"`
	SerperAPIKey                         *string   `json:"SERPER_API_KEY,omitempty"`
	SerphouseAPIKey                      *string   `json:"SERPHOUSE_API_KEY,omitempty"`
	SerphouseDomain                      *string   `json:"SERPHOUSE_DOMAIN,omitempty"`
	SerplyAPIKey                         *string   `json:"SERPLY_API_KEY,omitempty"`
	DdgsBackend                          *string   `json:"DDGS_BACKEND,omitempty"`
	TavilyAPIKey                         *string   `json:"TAVILY_API_KEY,omitempty"`
	SearchapiAPIKey                      *string   `json:"SEARCHAPI_API_KEY,omitempty"`
	SearchapiEngine                      *string   `json:"SEARCHAPI_ENGINE,omitempty"`
	SerpapiAPIKey                        *string   `json:"SERPAPI_API_KEY,omitempty"`
	SerpapiEngine                        *string   `json:"SERPAPI_ENGINE,omitempty"`
	JinaAPIKey                           *string   `json:"JINA_API_KEY,omitempty"`
	JinaAPIBaseURL                       *string   `json:"JINA_API_BASE_URL,omitempty"`
	BingSearchV7Endpoint                 *string   `json:"BING_SEARCH_V7_ENDPOINT,omitempty"`
	BingSearchV7SubscriptionKey          *string   `json:"BING_SEARCH_V7_SUBSCRIPTION_KEY,omitempty"`
	ExaAPIKey                            *string   `json:"EXA_API_KEY,omitempty"`
	PerplexityAPIKey                     *string   `json:"PERPLEXITY_API_KEY,omitempty"`
	PerplexityModel                      *string   `json:"PERPLEXITY_MODEL,omitempty"`
	PerplexitySearchContextUsage         *string   `json:"PERPLEXITY_SEARCH_CONTEXT_USAGE,omitempty"`
	PerplexitySearchAPIURL               *string   `json:"PERPLEXITY_SEARCH_API_URL,omitempty"`
	MicrosoftWebIQAPIBaseURL             *string   `json:"MICROSOFT_WEB_IQ_API_BASE_URL,omitempty"`
	MicrosoftWebIQAPIKey                 *string   `json:"MICROSOFT_WEB_IQ_API_KEY,omitempty"`
	MicrosoftWebIQLanguage               *string   `json:"MICROSOFT_WEB_IQ_LANGUAGE,omitempty"`
	SougouAPISID                         *string   `json:"SOUGOU_API_SID,omitempty"`
	SougouAPISK                          *string   `json:"SOUGOU_API_SK,omitempty"`
	WebLoaderEngine                      *string   `json:"WEB_LOADER_ENGINE,omitempty"`
	WebLoaderTimeout                     *string   `json:"WEB_LOADER_TIMEOUT,omitempty"`
	EnableWebLoaderSSLVerification       *bool     `json:"ENABLE_WEB_LOADER_SSL_VERIFICATION,omitempty"`
	PlaywrightWSURL                      *string   `json:"PLAYWRIGHT_WS_URL,omitempty"`
	PlaywrightTimeout                    *int64    `json:"PLAYWRIGHT_TIMEOUT,omitempty"`
	FirecrawlAPIKey                      *string   `json:"FIRECRAWL_API_KEY,omitempty"`
	FirecrawlAPIBaseURL                  *string   `json:"FIRECRAWL_API_BASE_URL,omitempty"`
	FirecrawlTimeout                     *string   `json:"FIRECRAWL_TIMEOUT,omitempty"`
	TavilyExtractDepth                   *string   `json:"TAVILY_EXTRACT_DEPTH,omitempty"`
	ExternalWebSearchURL                 *string   `json:"EXTERNAL_WEB_SEARCH_URL,omitempty"`
	ExternalWebSearchAPIKey              *string   `json:"EXTERNAL_WEB_SEARCH_API_KEY,omitempty"`
	ExternalWebLoaderURL                 *string   `json:"EXTERNAL_WEB_LOADER_URL,omitempty"`
	ExternalWebLoaderAPIKey              *string   `json:"EXTERNAL_WEB_LOADER_API_KEY,omitempty"`
	YoutubeLoaderLanguage                *[]string `json:"YOUTUBE_LOADER_LANGUAGE,omitempty"`
	YoutubeLoaderProxyURL                *string   `json:"YOUTUBE_LOADER_PROXY_URL,omitempty"`
	YandexWebSearchURL                   *string   `json:"YANDEX_WEB_SEARCH_URL,omitempty"`
	YandexWebSearchAPIKey                *string   `json:"YANDEX_WEB_SEARCH_API_KEY,omitempty"`
	YandexWebSearchConfig                *string   `json:"YANDEX_WEB_SEARCH_CONFIG,omitempty"`
	YoucomAPIKey                         *string   `json:"YOUCOM_API_KEY,omitempty"`
	LinkupAPIKey                         *string   `json:"LINKUP_API_KEY,omitempty"`
	LinkupSearchParams                   any       `json:"LINKUP_SEARCH_PARAMS,omitempty"`
}

// GetRAGEmbeddingConfig retrieves the embedding engine config.
func (c *Client) GetRAGEmbeddingConfig(ctx context.Context) (*RAGEmbeddingConfigForm, error) {
	var resp RAGEmbeddingConfigForm
	if err := c.do(ctx, http.MethodGet, "retrieval/embedding", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetRAGEmbeddingConfig updates the embedding engine config. The route loads
// the new embedding model into the Open WebUI process and unloads the previous
// one, so a call that switches engines does real work and can be slow.
func (c *Client) SetRAGEmbeddingConfig(ctx context.Context, form RAGEmbeddingConfigForm) (*RAGEmbeddingConfigForm, error) {
	var resp RAGEmbeddingConfigForm
	if err := c.do(ctx, http.MethodPost, "retrieval/embedding/update", nil, form, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetRAGConfig retrieves the retrieval config.
func (c *Client) GetRAGConfig(ctx context.Context) (*RAGConfigForm, error) {
	var resp RAGConfigForm
	if err := c.do(ctx, http.MethodGet, "retrieval/config", nil, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetRAGConfig updates the retrieval config and reads it back. The update
// route's own response omits five keys the GET returns, so the response body
// cannot serve as the new state.
func (c *Client) SetRAGConfig(ctx context.Context, form RAGConfigForm) (*RAGConfigForm, error) {
	if err := c.do(ctx, http.MethodPost, "retrieval/config/update", nil, form, nil); err != nil {
		return nil, err
	}

	return c.GetRAGConfig(ctx)
}

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func init() {
	registeredResources = append(registeredResources, NewRAGConfigResource)
}

var _ resource.Resource = &ragConfigResource{}
var _ resource.ResourceWithConfigure = &ragConfigResource{}
var _ resource.ResourceWithImportState = &ragConfigResource{}

// ragConfigResource manages document ingestion, retrieval, and web search.
//
// Four keys of RETRIEVAL_CONFIG_KEYS are loaded but unreachable through this
// API: AZURE_AI_SEARCH_API_KEY, AZURE_AI_SEARCH_ENDPOINT,
// AZURE_AI_SEARCH_INDEX_NAME, and TIKTOKEN_ENCODING_NAME. Neither route reads
// or writes them, so openwebui_config_import is the only way to set them. Two
// more keys in that map belong to other resources: USER_PERMISSIONS to
// openwebui_default_user_permissions and WEBUI_URL to openwebui_admin_config.
type ragConfigResource struct {
	client *client.Client
}

// ragConfigModel mirrors ConfigForm in backend/open_webui/routers/retrieval.py.
type ragConfigModel struct {
	ID                                       types.String       `tfsdk:"id"`
	RAGTemplate                              types.String       `tfsdk:"rag_template"`
	TopK                                     types.Int64        `tfsdk:"top_k"`
	BypassEmbeddingAndRetrieval              types.Bool         `tfsdk:"bypass_embedding_and_retrieval"`
	RAGFullContext                           types.Bool         `tfsdk:"rag_full_context"`
	EnableRAGHybridSearch                    types.Bool         `tfsdk:"enable_rag_hybrid_search"`
	EnableRAGHybridSearchEnrichedTexts       types.Bool         `tfsdk:"enable_rag_hybrid_search_enriched_texts"`
	TopKReranker                             types.Int64        `tfsdk:"top_k_reranker"`
	RelevanceThreshold                       types.Float64      `tfsdk:"relevance_threshold"`
	HybridBM25Weight                         types.Float64      `tfsdk:"hybrid_bm25_weight"`
	ContentExtractionEngine                  types.String       `tfsdk:"content_extraction_engine"`
	ContentExtractionSupportedMediaMimeTypes types.List         `tfsdk:"content_extraction_supported_media_mime_types"`
	PDFExtractImages                         types.Bool         `tfsdk:"pdf_extract_images"`
	PDFLoaderMode                            types.String       `tfsdk:"pdf_loader_mode"`
	DatalabMarkerAPIKey                      types.String       `tfsdk:"datalab_marker_api_key"`
	DatalabMarkerAPIBaseURL                  types.String       `tfsdk:"datalab_marker_api_base_url"`
	DatalabMarkerAdditionalConfig            types.String       `tfsdk:"datalab_marker_additional_config"`
	DatalabMarkerSkipCache                   types.Bool         `tfsdk:"datalab_marker_skip_cache"`
	DatalabMarkerForceOCR                    types.Bool         `tfsdk:"datalab_marker_force_ocr"`
	DatalabMarkerPaginate                    types.Bool         `tfsdk:"datalab_marker_paginate"`
	DatalabMarkerStripExistingOCR            types.Bool         `tfsdk:"datalab_marker_strip_existing_ocr"`
	DatalabMarkerDisableImageExtraction      types.Bool         `tfsdk:"datalab_marker_disable_image_extraction"`
	DatalabMarkerFormatLines                 types.Bool         `tfsdk:"datalab_marker_format_lines"`
	DatalabMarkerUseLLM                      types.Bool         `tfsdk:"datalab_marker_use_llm"`
	DatalabMarkerOutputFormat                types.String       `tfsdk:"datalab_marker_output_format"`
	ExternalDocumentLoaderURL                types.String       `tfsdk:"external_document_loader_url"`
	ExternalDocumentLoaderAPIKey             types.String       `tfsdk:"external_document_loader_api_key"`
	ExternalDocumentLoaderHeaders            types.String       `tfsdk:"external_document_loader_headers"`
	TikaServerURL                            types.String       `tfsdk:"tika_server_url"`
	DoclingServerURL                         types.String       `tfsdk:"docling_server_url"`
	DoclingAPIKey                            types.String       `tfsdk:"docling_api_key"`
	DoclingParams                            types.String       `tfsdk:"docling_params"`
	DocumentIntelligenceEndpoint             types.String       `tfsdk:"document_intelligence_endpoint"`
	DocumentIntelligenceKey                  types.String       `tfsdk:"document_intelligence_key"`
	DocumentIntelligenceModel                types.String       `tfsdk:"document_intelligence_model"`
	MistralOCRAPIBaseURL                     types.String       `tfsdk:"mistral_ocr_api_base_url"`
	MistralOCRAPIKey                         types.String       `tfsdk:"mistral_ocr_api_key"`
	MistralOCRUseBase64                      types.Bool         `tfsdk:"mistral_ocr_use_base64"`
	PaddleocrVLBaseURL                       types.String       `tfsdk:"paddleocr_vl_base_url"`
	PaddleocrVLToken                         types.String       `tfsdk:"paddleocr_vl_token"`
	MineruAPIMode                            types.String       `tfsdk:"mineru_api_mode"`
	MineruAPIURL                             types.String       `tfsdk:"mineru_api_url"`
	MineruAPIKey                             types.String       `tfsdk:"mineru_api_key"`
	MineruAPITimeout                         types.Int64        `tfsdk:"mineru_api_timeout"`
	MineruParams                             types.String       `tfsdk:"mineru_params"`
	MineruFileExtensions                     types.List         `tfsdk:"mineru_file_extensions"`
	RAGRerankingModel                        types.String       `tfsdk:"rag_reranking_model"`
	RAGRerankingEngine                       types.String       `tfsdk:"rag_reranking_engine"`
	RAGRerankingBatchSize                    types.Int64        `tfsdk:"rag_reranking_batch_size"`
	RAGExternalRerankerURL                   types.String       `tfsdk:"rag_external_reranker_url"`
	RAGExternalRerankerAPIKey                types.String       `tfsdk:"rag_external_reranker_api_key"`
	RAGExternalRerankerTimeout               types.String       `tfsdk:"rag_external_reranker_timeout"`
	TextSplitter                             types.String       `tfsdk:"text_splitter"`
	RAGTokenizerModel                        types.String       `tfsdk:"rag_tokenizer_model"`
	EnableMarkdownHeaderTextSplitter         types.Bool         `tfsdk:"enable_markdown_header_text_splitter"`
	ChunkSize                                types.Int64        `tfsdk:"chunk_size"`
	ChunkMinSizeTarget                       types.Int64        `tfsdk:"chunk_min_size_target"`
	ChunkOverlap                             types.Int64        `tfsdk:"chunk_overlap"`
	FileMaxSize                              types.String       `tfsdk:"file_max_size"`
	FileMaxCount                             types.String       `tfsdk:"file_max_count"`
	FileImageCompressionWidth                types.String       `tfsdk:"file_image_compression_width"`
	FileImageCompressionHeight               types.String       `tfsdk:"file_image_compression_height"`
	AllowedFileExtensions                    types.List         `tfsdk:"allowed_file_extensions"`
	EnableGoogleDriveIntegration             types.Bool         `tfsdk:"enable_google_drive_integration"`
	EnableOneDriveIntegration                types.Bool         `tfsdk:"enable_onedrive_integration"`
	Web                                      *ragWebConfigModel `tfsdk:"web"`
}

// ragWebConfigModel mirrors WebConfig in backend/open_webui/routers/retrieval.py,
// less YOUTUBE_LOADER_TRANSLATION, which has no storage key.
type ragWebConfigModel struct {
	EnableWebSearch                      types.Bool   `tfsdk:"enable_web_search"`
	EnableWebSearchConfirmation          types.Bool   `tfsdk:"enable_web_search_confirmation"`
	WebSearchConfirmationContent         types.String `tfsdk:"web_search_confirmation_content"`
	WebSearchEngine                      types.String `tfsdk:"web_search_engine"`
	WebSearchTrustEnv                    types.Bool   `tfsdk:"web_search_trust_env"`
	WebSearchResultCount                 types.Int64  `tfsdk:"web_search_result_count"`
	WebSearchConcurrentRequests          types.Int64  `tfsdk:"web_search_concurrent_requests"`
	WebSearchDomainFilterList            types.List   `tfsdk:"web_search_domain_filter_list"`
	WebFetchMaxContentLength             types.Int64  `tfsdk:"web_fetch_max_content_length"`
	WebLoaderConcurrentRequests          types.Int64  `tfsdk:"web_loader_concurrent_requests"`
	BypassWebSearchEmbeddingAndRetrieval types.Bool   `tfsdk:"bypass_web_search_embedding_and_retrieval"`
	BypassWebSearchWebLoader             types.Bool   `tfsdk:"bypass_web_search_web_loader"`
	OllamaCloudWebSearchAPIKey           types.String `tfsdk:"ollama_cloud_web_search_api_key"`
	SearxngQueryURL                      types.String `tfsdk:"searxng_query_url"`
	SearxngLanguage                      types.String `tfsdk:"searxng_language"`
	OpenserpBaseURL                      types.String `tfsdk:"openserp_base_url"`
	YacyQueryURL                         types.String `tfsdk:"yacy_query_url"`
	YacyUsername                         types.String `tfsdk:"yacy_username"`
	YacyPassword                         types.String `tfsdk:"yacy_password"`
	GooglePSEAPIKey                      types.String `tfsdk:"google_pse_api_key"`
	GooglePSEEngineID                    types.String `tfsdk:"google_pse_engine_id"`
	BraveSearchAPIKey                    types.String `tfsdk:"brave_search_api_key"`
	BraveSearchContextTokens             types.Int64  `tfsdk:"brave_search_context_tokens"`
	KagiSearchAPIKey                     types.String `tfsdk:"kagi_search_api_key"`
	MojeekSearchAPIKey                   types.String `tfsdk:"mojeek_search_api_key"`
	BochaSearchAPIKey                    types.String `tfsdk:"bocha_search_api_key"`
	SerpstackAPIKey                      types.String `tfsdk:"serpstack_api_key"`
	SerpstackHTTPS                       types.Bool   `tfsdk:"serpstack_https"`
	SerperAPIKey                         types.String `tfsdk:"serper_api_key"`
	SerphouseAPIKey                      types.String `tfsdk:"serphouse_api_key"`
	SerphouseDomain                      types.String `tfsdk:"serphouse_domain"`
	SerplyAPIKey                         types.String `tfsdk:"serply_api_key"`
	DdgsBackend                          types.String `tfsdk:"ddgs_backend"`
	TavilyAPIKey                         types.String `tfsdk:"tavily_api_key"`
	SearchapiAPIKey                      types.String `tfsdk:"searchapi_api_key"`
	SearchapiEngine                      types.String `tfsdk:"searchapi_engine"`
	SerpapiAPIKey                        types.String `tfsdk:"serpapi_api_key"`
	SerpapiEngine                        types.String `tfsdk:"serpapi_engine"`
	JinaAPIKey                           types.String `tfsdk:"jina_api_key"`
	JinaAPIBaseURL                       types.String `tfsdk:"jina_api_base_url"`
	BingSearchV7Endpoint                 types.String `tfsdk:"bing_search_v7_endpoint"`
	BingSearchV7SubscriptionKey          types.String `tfsdk:"bing_search_v7_subscription_key"`
	ExaAPIKey                            types.String `tfsdk:"exa_api_key"`
	PerplexityAPIKey                     types.String `tfsdk:"perplexity_api_key"`
	PerplexityModel                      types.String `tfsdk:"perplexity_model"`
	PerplexitySearchContextUsage         types.String `tfsdk:"perplexity_search_context_usage"`
	PerplexitySearchAPIURL               types.String `tfsdk:"perplexity_search_api_url"`
	MicrosoftWebIQAPIBaseURL             types.String `tfsdk:"microsoft_web_iq_api_base_url"`
	MicrosoftWebIQAPIKey                 types.String `tfsdk:"microsoft_web_iq_api_key"`
	MicrosoftWebIQLanguage               types.String `tfsdk:"microsoft_web_iq_language"`
	SougouAPISID                         types.String `tfsdk:"sougou_api_sid"`
	SougouAPISK                          types.String `tfsdk:"sougou_api_sk"`
	WebLoaderEngine                      types.String `tfsdk:"web_loader_engine"`
	WebLoaderTimeout                     types.String `tfsdk:"web_loader_timeout"`
	EnableWebLoaderSSLVerification       types.Bool   `tfsdk:"enable_web_loader_ssl_verification"`
	PlaywrightWSURL                      types.String `tfsdk:"playwright_ws_url"`
	PlaywrightTimeout                    types.Int64  `tfsdk:"playwright_timeout"`
	FirecrawlAPIKey                      types.String `tfsdk:"firecrawl_api_key"`
	FirecrawlAPIBaseURL                  types.String `tfsdk:"firecrawl_api_base_url"`
	FirecrawlTimeout                     types.String `tfsdk:"firecrawl_timeout"`
	TavilyExtractDepth                   types.String `tfsdk:"tavily_extract_depth"`
	ExternalWebSearchURL                 types.String `tfsdk:"external_web_search_url"`
	ExternalWebSearchAPIKey              types.String `tfsdk:"external_web_search_api_key"`
	ExternalWebLoaderURL                 types.String `tfsdk:"external_web_loader_url"`
	ExternalWebLoaderAPIKey              types.String `tfsdk:"external_web_loader_api_key"`
	YoutubeLoaderLanguage                types.List   `tfsdk:"youtube_loader_language"`
	YoutubeLoaderProxyURL                types.String `tfsdk:"youtube_loader_proxy_url"`
	YandexWebSearchURL                   types.String `tfsdk:"yandex_web_search_url"`
	YandexWebSearchAPIKey                types.String `tfsdk:"yandex_web_search_api_key"`
	YandexWebSearchConfig                types.String `tfsdk:"yandex_web_search_config"`
	YoucomAPIKey                         types.String `tfsdk:"youcom_api_key"`
	LinkupAPIKey                         types.String `tfsdk:"linkup_api_key"`
	LinkupSearchParams                   types.String `tfsdk:"linkup_search_params"`
}

// NewRAGConfigResource constructs a new RAG config resource.
func NewRAGConfigResource() resource.Resource {
	return &ragConfigResource{}
}

// Metadata sets the resource type name.
func (r *ragConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rag_config"
}

// Schema defines the RAG config schema.
func (r *ragConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's retrieval settings: document loaders, chunking, reranking, file limits, and web search.\n\n" +
			"The top-level keys keep their stored value when a request leaves them out, but every key of the `web` block is written on " +
			"every write. This resource therefore reads the current configuration and overlays the attributes the plan names, so a " +
			"partial configuration never clears a key it does not mention.\n\n" +
			"`POST /api/v1/retrieval/config/update` answers with a response that omits five of the keys the GET returns, so this " +
			"resource reads the configuration back after every write.\n\n" +
			"Four keys are loaded but unreachable through this router: `AZURE_AI_SEARCH_API_KEY`, `AZURE_AI_SEARCH_ENDPOINT`, " +
			"`AZURE_AI_SEARCH_INDEX_NAME`, and `TIKTOKEN_ENCODING_NAME`. Set them with `openwebui_config_import`.\n\n" +
			"`YOUTUBE_LOADER_TRANSLATION` has no attribute here. It is in `WebConfig` but not in `RETRIEVAL_CONFIG_KEYS`, so Open WebUI " +
			"holds it in process memory and loses it on restart.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier of this singleton resource. Always `rag`.",
				MarkdownDescription: "Identifier of this singleton resource. Always `rag`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"rag_template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Prompt template that wraps retrieved context. Stored as `rag.template`.",
				MarkdownDescription: "Prompt template that wraps retrieved context. Stored as `rag.template`.",
			},
			"top_k": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `TOP_K`. Stored as `rag.top_k`.",
				MarkdownDescription: "Open WebUI setting `TOP_K`. Stored as `rag.top_k`.",
			},
			"bypass_embedding_and_retrieval": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `BYPASS_EMBEDDING_AND_RETRIEVAL`. Stored as `rag.bypass_embedding_and_retrieval`.",
				MarkdownDescription: "Open WebUI setting `BYPASS_EMBEDDING_AND_RETRIEVAL`. Stored as `rag.bypass_embedding_and_retrieval`.",
			},
			"rag_full_context": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_FULL_CONTEXT`. Stored as `rag.full_context`.",
				MarkdownDescription: "Open WebUI setting `RAG_FULL_CONTEXT`. Stored as `rag.full_context`.",
			},
			"enable_rag_hybrid_search": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_RAG_HYBRID_SEARCH`. Stored as `rag.enable_hybrid_search`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_RAG_HYBRID_SEARCH`. Stored as `rag.enable_hybrid_search`.",
			},
			"enable_rag_hybrid_search_enriched_texts": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS`. Stored as `rag.enable_hybrid_search_enriched_texts`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS`. Stored as `rag.enable_hybrid_search_enriched_texts`.",
			},
			"top_k_reranker": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `TOP_K_RERANKER`. Stored as `rag.top_k_reranker`.",
				MarkdownDescription: "Open WebUI setting `TOP_K_RERANKER`. Stored as `rag.top_k_reranker`.",
			},
			"relevance_threshold": schema.Float64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RELEVANCE_THRESHOLD`. Stored as `rag.relevance_threshold`.",
				MarkdownDescription: "Open WebUI setting `RELEVANCE_THRESHOLD`. Stored as `rag.relevance_threshold`.",
			},
			"hybrid_bm25_weight": schema.Float64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `HYBRID_BM25_WEIGHT`. Stored as `rag.hybrid_bm25_weight`.",
				MarkdownDescription: "Open WebUI setting `HYBRID_BM25_WEIGHT`. Stored as `rag.hybrid_bm25_weight`.",
			},
			"content_extraction_engine": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `CONTENT_EXTRACTION_ENGINE`. Stored as `rag.content_extraction_engine`.",
				MarkdownDescription: "Open WebUI setting `CONTENT_EXTRACTION_ENGINE`. Stored as `rag.content_extraction_engine`.",
			},
			"content_extraction_supported_media_mime_types": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES`. Stored as `rag.content_extraction.supported_media_mime_types`.",
				MarkdownDescription: "Open WebUI setting `CONTENT_EXTRACTION_SUPPORTED_MEDIA_MIME_TYPES`. Stored as `rag.content_extraction.supported_media_mime_types`.",
			},
			"pdf_extract_images": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `PDF_EXTRACT_IMAGES`. Stored as `rag.pdf_extract_images`.",
				MarkdownDescription: "Open WebUI setting `PDF_EXTRACT_IMAGES`. Stored as `rag.pdf_extract_images`.",
			},
			"pdf_loader_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `PDF_LOADER_MODE`. Stored as `rag.pdf_loader_mode`.",
				MarkdownDescription: "Open WebUI setting `PDF_LOADER_MODE`. Stored as `rag.pdf_loader_mode`.",
			},
			"datalab_marker_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `DATALAB_MARKER_API_KEY`. Stored as `rag.datalab_marker_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_API_KEY`. Stored as `rag.datalab_marker_api_key`. Sensitive.",
			},
			"datalab_marker_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_API_BASE_URL`. Stored as `rag.datalab_marker_api_base_url`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_API_BASE_URL`. Stored as `rag.datalab_marker_api_base_url`.",
			},
			"datalab_marker_additional_config": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_ADDITIONAL_CONFIG`. Stored as `rag.datalab_marker_additional_config`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_ADDITIONAL_CONFIG`. Stored as `rag.datalab_marker_additional_config`.",
			},
			"datalab_marker_skip_cache": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_SKIP_CACHE`. Stored as `rag.datalab_marker_skip_cache`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_SKIP_CACHE`. Stored as `rag.datalab_marker_skip_cache`.",
			},
			"datalab_marker_force_ocr": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_FORCE_OCR`. Stored as `rag.datalab_marker_force_ocr`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_FORCE_OCR`. Stored as `rag.datalab_marker_force_ocr`.",
			},
			"datalab_marker_paginate": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_PAGINATE`. Stored as `rag.datalab_marker_paginate`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_PAGINATE`. Stored as `rag.datalab_marker_paginate`.",
			},
			"datalab_marker_strip_existing_ocr": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_STRIP_EXISTING_OCR`. Stored as `rag.datalab_marker_strip_existing_ocr`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_STRIP_EXISTING_OCR`. Stored as `rag.datalab_marker_strip_existing_ocr`.",
			},
			"datalab_marker_disable_image_extraction": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION`. Stored as `rag.datalab_marker_disable_image_extraction`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_DISABLE_IMAGE_EXTRACTION`. Stored as `rag.datalab_marker_disable_image_extraction`.",
			},
			"datalab_marker_format_lines": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_FORMAT_LINES`. Stored as `rag.datalab_marker_format_lines`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_FORMAT_LINES`. Stored as `rag.datalab_marker_format_lines`.",
			},
			"datalab_marker_use_llm": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_USE_LLM`. Stored as `rag.datalab_marker_use_llm`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_USE_LLM`. Stored as `rag.datalab_marker_use_llm`.",
			},
			"datalab_marker_output_format": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DATALAB_MARKER_OUTPUT_FORMAT`. Stored as `rag.datalab_marker_output_format`.",
				MarkdownDescription: "Open WebUI setting `DATALAB_MARKER_OUTPUT_FORMAT`. Stored as `rag.datalab_marker_output_format`.",
			},
			"external_document_loader_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `EXTERNAL_DOCUMENT_LOADER_URL`. Stored as `rag.external_document_loader_url`.",
				MarkdownDescription: "Open WebUI setting `EXTERNAL_DOCUMENT_LOADER_URL`. Stored as `rag.external_document_loader_url`.",
			},
			"external_document_loader_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `EXTERNAL_DOCUMENT_LOADER_API_KEY`. Stored as `rag.external_document_loader_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `EXTERNAL_DOCUMENT_LOADER_API_KEY`. Stored as `rag.external_document_loader_api_key`. Sensitive.",
			},
			"external_document_loader_headers": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "JSON object of headers sent to the external document loader. Carries the loader's credentials. Stored as `rag.external_document_loader_headers`. Sensitive.",
				MarkdownDescription: "JSON object of headers sent to the external document loader. Carries the loader's credentials. Stored as `rag.external_document_loader_headers`. Sensitive.",
			},
			"tika_server_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `TIKA_SERVER_URL`. Stored as `rag.tika_server_url`.",
				MarkdownDescription: "Open WebUI setting `TIKA_SERVER_URL`. Stored as `rag.tika_server_url`.",
			},
			"docling_server_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DOCLING_SERVER_URL`. Stored as `rag.docling_server_url`.",
				MarkdownDescription: "Open WebUI setting `DOCLING_SERVER_URL`. Stored as `rag.docling_server_url`.",
			},
			"docling_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `DOCLING_API_KEY`. Stored as `rag.docling_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `DOCLING_API_KEY`. Stored as `rag.docling_api_key`. Sensitive.",
			},
			"docling_params": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of extra parameters passed to Docling. Stored as `rag.docling_params`.",
				MarkdownDescription: "JSON object of extra parameters passed to Docling. Stored as `rag.docling_params`.",
			},
			"document_intelligence_endpoint": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DOCUMENT_INTELLIGENCE_ENDPOINT`. Stored as `rag.document_intelligence_endpoint`.",
				MarkdownDescription: "Open WebUI setting `DOCUMENT_INTELLIGENCE_ENDPOINT`. Stored as `rag.document_intelligence_endpoint`.",
			},
			"document_intelligence_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `DOCUMENT_INTELLIGENCE_KEY`. Stored as `rag.document_intelligence_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `DOCUMENT_INTELLIGENCE_KEY`. Stored as `rag.document_intelligence_key`. Sensitive.",
			},
			"document_intelligence_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `DOCUMENT_INTELLIGENCE_MODEL`. Stored as `rag.document_intelligence_model`.",
				MarkdownDescription: "Open WebUI setting `DOCUMENT_INTELLIGENCE_MODEL`. Stored as `rag.document_intelligence_model`.",
			},
			"mistral_ocr_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `MISTRAL_OCR_API_BASE_URL`. Stored as `rag.mistral_ocr_api_base_url`.",
				MarkdownDescription: "Open WebUI setting `MISTRAL_OCR_API_BASE_URL`. Stored as `rag.mistral_ocr_api_base_url`.",
			},
			"mistral_ocr_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `MISTRAL_OCR_API_KEY`. Stored as `rag.mistral_ocr_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `MISTRAL_OCR_API_KEY`. Stored as `rag.mistral_ocr_api_key`. Sensitive.",
			},
			"mistral_ocr_use_base64": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `MISTRAL_OCR_USE_BASE64`. Stored as `rag.mistral_ocr_use_base64`.",
				MarkdownDescription: "Open WebUI setting `MISTRAL_OCR_USE_BASE64`. Stored as `rag.mistral_ocr_use_base64`.",
			},
			"paddleocr_vl_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `PADDLEOCR_VL_BASE_URL`. Stored as `rag.paddleocr_vl_base_url`.",
				MarkdownDescription: "Open WebUI setting `PADDLEOCR_VL_BASE_URL`. Stored as `rag.paddleocr_vl_base_url`.",
			},
			"paddleocr_vl_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `PADDLEOCR_VL_TOKEN`. Stored as `rag.paddleocr_vl_token`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `PADDLEOCR_VL_TOKEN`. Stored as `rag.paddleocr_vl_token`. Sensitive.",
			},
			"mineru_api_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `MINERU_API_MODE`. Stored as `rag.mineru_api_mode`.",
				MarkdownDescription: "Open WebUI setting `MINERU_API_MODE`. Stored as `rag.mineru_api_mode`.",
			},
			"mineru_api_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `MINERU_API_URL`. Stored as `rag.mineru_api_url`.",
				MarkdownDescription: "Open WebUI setting `MINERU_API_URL`. Stored as `rag.mineru_api_url`.",
			},
			"mineru_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `MINERU_API_KEY`. Stored as `rag.mineru_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `MINERU_API_KEY`. Stored as `rag.mineru_api_key`. Sensitive.",
			},
			"mineru_api_timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "MinerU request timeout in seconds. Open WebUI ships the default as a string and reads it back as either a string or a number. Stored as `rag.mineru_api_timeout`.",
				MarkdownDescription: "MinerU request timeout in seconds. Open WebUI ships the default as a string and reads it back as either a string or a number. Stored as `rag.mineru_api_timeout`.",
			},
			"mineru_params": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of extra parameters passed to MinerU. Stored as `rag.mineru_params`.",
				MarkdownDescription: "JSON object of extra parameters passed to MinerU. Stored as `rag.mineru_params`.",
			},
			"mineru_file_extensions": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `MINERU_FILE_EXTENSIONS`. Stored as `rag.mineru_file_extensions`.",
				MarkdownDescription: "Open WebUI setting `MINERU_FILE_EXTENSIONS`. Stored as `rag.mineru_file_extensions`.",
			},
			"rag_reranking_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_RERANKING_MODEL`. Stored as `rag.reranking_model`.",
				MarkdownDescription: "Open WebUI setting `RAG_RERANKING_MODEL`. Stored as `rag.reranking_model`.",
			},
			"rag_reranking_engine": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_RERANKING_ENGINE`. Stored as `rag.reranking_engine`.",
				MarkdownDescription: "Open WebUI setting `RAG_RERANKING_ENGINE`. Stored as `rag.reranking_engine`.",
			},
			"rag_reranking_batch_size": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_RERANKING_BATCH_SIZE`. Stored as `rag.reranking_batch_size`.",
				MarkdownDescription: "Open WebUI setting `RAG_RERANKING_BATCH_SIZE`. Stored as `rag.reranking_batch_size`.",
			},
			"rag_external_reranker_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_EXTERNAL_RERANKER_URL`. Stored as `rag.external_reranker_url`.",
				MarkdownDescription: "Open WebUI setting `RAG_EXTERNAL_RERANKER_URL`. Stored as `rag.external_reranker_url`.",
			},
			"rag_external_reranker_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `RAG_EXTERNAL_RERANKER_API_KEY`. Stored as `rag.external_reranker_api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `RAG_EXTERNAL_RERANKER_API_KEY`. Stored as `rag.external_reranker_api_key`. Sensitive.",
			},
			"rag_external_reranker_timeout": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_EXTERNAL_RERANKER_TIMEOUT`. Stored as `rag.external_reranker_timeout`.",
				MarkdownDescription: "Open WebUI setting `RAG_EXTERNAL_RERANKER_TIMEOUT`. Stored as `rag.external_reranker_timeout`.",
			},
			"text_splitter": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `TEXT_SPLITTER`. Stored as `rag.text_splitter`.",
				MarkdownDescription: "Open WebUI setting `TEXT_SPLITTER`. Stored as `rag.text_splitter`.",
			},
			"rag_tokenizer_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `RAG_TOKENIZER_MODEL`. Stored as `rag.tokenizer_model`.",
				MarkdownDescription: "Open WebUI setting `RAG_TOKENIZER_MODEL`. Stored as `rag.tokenizer_model`.",
			},
			"enable_markdown_header_text_splitter": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER`. Stored as `rag.enable_markdown_header_text_splitter`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_MARKDOWN_HEADER_TEXT_SPLITTER`. Stored as `rag.enable_markdown_header_text_splitter`.",
			},
			"chunk_size": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `CHUNK_SIZE`. Stored as `rag.chunk_size`.",
				MarkdownDescription: "Open WebUI setting `CHUNK_SIZE`. Stored as `rag.chunk_size`.",
			},
			"chunk_min_size_target": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `CHUNK_MIN_SIZE_TARGET`. Stored as `rag.chunk_min_size_target`.",
				MarkdownDescription: "Open WebUI setting `CHUNK_MIN_SIZE_TARGET`. Stored as `rag.chunk_min_size_target`.",
			},
			"chunk_overlap": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `CHUNK_OVERLAP`. Stored as `rag.chunk_overlap`.",
				MarkdownDescription: "Open WebUI setting `CHUNK_OVERLAP`. Stored as `rag.chunk_overlap`.",
			},
			"file_max_size": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Largest upload Open WebUI accepts, in megabytes. The empty string removes the limit. Stored as `rag.file.max_size`.",
				MarkdownDescription: "Largest upload Open WebUI accepts, in megabytes. The empty string removes the limit. Stored as `rag.file.max_size`.",
			},
			"file_max_count": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Largest number of files one chat message may carry. The empty string removes the limit. Stored as `rag.file.max_count`.",
				MarkdownDescription: "Largest number of files one chat message may carry. The empty string removes the limit. Stored as `rag.file.max_count`.",
			},
			"file_image_compression_width": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Width an uploaded image is compressed to. The empty string turns compression off. Stored as `file.image_compression_width`.",
				MarkdownDescription: "Width an uploaded image is compressed to. The empty string turns compression off. Stored as `file.image_compression_width`.",
			},
			"file_image_compression_height": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Height an uploaded image is compressed to. The empty string turns compression off. Stored as `file.image_compression_height`.",
				MarkdownDescription: "Height an uploaded image is compressed to. The empty string turns compression off. Stored as `file.image_compression_height`.",
			},
			"allowed_file_extensions": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ALLOWED_FILE_EXTENSIONS`. Stored as `rag.file.allowed_extensions`.",
				MarkdownDescription: "Open WebUI setting `ALLOWED_FILE_EXTENSIONS`. Stored as `rag.file.allowed_extensions`.",
			},
			"enable_google_drive_integration": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_GOOGLE_DRIVE_INTEGRATION`. Stored as `google_drive.enable`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_GOOGLE_DRIVE_INTEGRATION`. Stored as `google_drive.enable`.",
			},
			"enable_onedrive_integration": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_ONEDRIVE_INTEGRATION`. Stored as `onedrive.enable`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_ONEDRIVE_INTEGRATION`. Stored as `onedrive.enable`.",
			},
			"web": schema.SingleNestedAttribute{
				Optional:            true,
				Description:         "Web search and web loader settings. Open WebUI writes every key of this block on every write.",
				MarkdownDescription: "Web search and web loader settings. Open WebUI writes every key of this block on every write.",
				Attributes: map[string]schema.Attribute{
					"enable_web_search": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `ENABLE_WEB_SEARCH`. Stored as `web.search.enable`.",
						MarkdownDescription: "Open WebUI setting `ENABLE_WEB_SEARCH`. Stored as `web.search.enable`.",
					},
					"enable_web_search_confirmation": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `ENABLE_WEB_SEARCH_CONFIRMATION`. Stored as `web.search.confirmation.enable`.",
						MarkdownDescription: "Open WebUI setting `ENABLE_WEB_SEARCH_CONFIRMATION`. Stored as `web.search.confirmation.enable`.",
					},
					"web_search_confirmation_content": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_SEARCH_CONFIRMATION_CONTENT`. Stored as `web.search.confirmation.content`.",
						MarkdownDescription: "Open WebUI setting `WEB_SEARCH_CONFIRMATION_CONTENT`. Stored as `web.search.confirmation.content`.",
					},
					"web_search_engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_SEARCH_ENGINE`. Stored as `web.search.engine`.",
						MarkdownDescription: "Open WebUI setting `WEB_SEARCH_ENGINE`. Stored as `web.search.engine`.",
					},
					"web_search_trust_env": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_SEARCH_TRUST_ENV`. Stored as `web.search.trust_env`.",
						MarkdownDescription: "Open WebUI setting `WEB_SEARCH_TRUST_ENV`. Stored as `web.search.trust_env`.",
					},
					"web_search_result_count": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_SEARCH_RESULT_COUNT`. Stored as `web.search.result_count`.",
						MarkdownDescription: "Open WebUI setting `WEB_SEARCH_RESULT_COUNT`. Stored as `web.search.result_count`.",
					},
					"web_search_concurrent_requests": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_SEARCH_CONCURRENT_REQUESTS`. Stored as `web.search.concurrent_requests`.",
						MarkdownDescription: "Open WebUI setting `WEB_SEARCH_CONCURRENT_REQUESTS`. Stored as `web.search.concurrent_requests`.",
					},
					"web_search_domain_filter_list": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Description:         "Domains web search results are restricted to. Stored as `web.search.domain.filter_list`.",
						MarkdownDescription: "Domains web search results are restricted to. Stored as `web.search.domain.filter_list`.",
					},
					"web_fetch_max_content_length": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_FETCH_MAX_CONTENT_LENGTH`. Stored as `web.fetch.max_content_length`.",
						MarkdownDescription: "Open WebUI setting `WEB_FETCH_MAX_CONTENT_LENGTH`. Stored as `web.fetch.max_content_length`.",
					},
					"web_loader_concurrent_requests": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_LOADER_CONCURRENT_REQUESTS`. Stored as `web.loader.concurrent_requests`.",
						MarkdownDescription: "Open WebUI setting `WEB_LOADER_CONCURRENT_REQUESTS`. Stored as `web.loader.concurrent_requests`.",
					},
					"bypass_web_search_embedding_and_retrieval": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL`. Stored as `web.search.bypass_embedding_and_retrieval`.",
						MarkdownDescription: "Open WebUI setting `BYPASS_WEB_SEARCH_EMBEDDING_AND_RETRIEVAL`. Stored as `web.search.bypass_embedding_and_retrieval`.",
					},
					"bypass_web_search_web_loader": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `BYPASS_WEB_SEARCH_WEB_LOADER`. Stored as `web.search.bypass_web_loader`.",
						MarkdownDescription: "Open WebUI setting `BYPASS_WEB_SEARCH_WEB_LOADER`. Stored as `web.search.bypass_web_loader`.",
					},
					"ollama_cloud_web_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `OLLAMA_CLOUD_WEB_SEARCH_API_KEY`. Stored as `web.search.ollama_cloud_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `OLLAMA_CLOUD_WEB_SEARCH_API_KEY`. Stored as `web.search.ollama_cloud_api_key`. Sensitive.",
					},
					"searxng_query_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SEARXNG_QUERY_URL`. Stored as `web.search.searxng_query_url`.",
						MarkdownDescription: "Open WebUI setting `SEARXNG_QUERY_URL`. Stored as `web.search.searxng_query_url`.",
					},
					"searxng_language": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SEARXNG_LANGUAGE`. Stored as `web.search.searxng_language`.",
						MarkdownDescription: "Open WebUI setting `SEARXNG_LANGUAGE`. Stored as `web.search.searxng_language`.",
					},
					"openserp_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `OPENSERP_BASE_URL`. Stored as `web.search.openserp_base_url`.",
						MarkdownDescription: "Open WebUI setting `OPENSERP_BASE_URL`. Stored as `web.search.openserp_base_url`.",
					},
					"yacy_query_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YACY_QUERY_URL`. Stored as `web.search.yacy_query_url`.",
						MarkdownDescription: "Open WebUI setting `YACY_QUERY_URL`. Stored as `web.search.yacy_query_url`.",
					},
					"yacy_username": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YACY_USERNAME`. Stored as `web.search.yacy_username`.",
						MarkdownDescription: "Open WebUI setting `YACY_USERNAME`. Stored as `web.search.yacy_username`.",
					},
					"yacy_password": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `YACY_PASSWORD`. Stored as `web.search.yacy_password`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `YACY_PASSWORD`. Stored as `web.search.yacy_password`. Sensitive.",
					},
					"google_pse_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `GOOGLE_PSE_API_KEY`. Stored as `web.search.google_pse_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `GOOGLE_PSE_API_KEY`. Stored as `web.search.google_pse_api_key`. Sensitive.",
					},
					"google_pse_engine_id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `GOOGLE_PSE_ENGINE_ID`. Stored as `web.search.google_pse_engine_id`.",
						MarkdownDescription: "Open WebUI setting `GOOGLE_PSE_ENGINE_ID`. Stored as `web.search.google_pse_engine_id`.",
					},
					"brave_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `BRAVE_SEARCH_API_KEY`. Stored as `web.search.brave_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `BRAVE_SEARCH_API_KEY`. Stored as `web.search.brave_search_api_key`. Sensitive.",
					},
					"brave_search_context_tokens": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `BRAVE_SEARCH_CONTEXT_TOKENS`. Stored as `web.search.brave_search_context_tokens`.",
						MarkdownDescription: "Open WebUI setting `BRAVE_SEARCH_CONTEXT_TOKENS`. Stored as `web.search.brave_search_context_tokens`.",
					},
					"kagi_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `KAGI_SEARCH_API_KEY`. Stored as `web.search.kagi_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `KAGI_SEARCH_API_KEY`. Stored as `web.search.kagi_search_api_key`. Sensitive.",
					},
					"mojeek_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `MOJEEK_SEARCH_API_KEY`. Stored as `web.search.mojeek_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `MOJEEK_SEARCH_API_KEY`. Stored as `web.search.mojeek_search_api_key`. Sensitive.",
					},
					"bocha_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `BOCHA_SEARCH_API_KEY`. Stored as `web.search.bocha_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `BOCHA_SEARCH_API_KEY`. Stored as `web.search.bocha_search_api_key`. Sensitive.",
					},
					"serpstack_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SERPSTACK_API_KEY`. Stored as `web.search.serpstack_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SERPSTACK_API_KEY`. Stored as `web.search.serpstack_api_key`. Sensitive.",
					},
					"serpstack_https": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SERPSTACK_HTTPS`. Stored as `web.search.serpstack_https`.",
						MarkdownDescription: "Open WebUI setting `SERPSTACK_HTTPS`. Stored as `web.search.serpstack_https`.",
					},
					"serper_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SERPER_API_KEY`. Stored as `web.search.serper_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SERPER_API_KEY`. Stored as `web.search.serper_api_key`. Sensitive.",
					},
					"serphouse_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SERPHOUSE_API_KEY`. Stored as `web.search.serphouse_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SERPHOUSE_API_KEY`. Stored as `web.search.serphouse_api_key`. Sensitive.",
					},
					"serphouse_domain": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SERPHOUSE_DOMAIN`. Stored as `web.search.serphouse_domain`.",
						MarkdownDescription: "Open WebUI setting `SERPHOUSE_DOMAIN`. Stored as `web.search.serphouse_domain`.",
					},
					"serply_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SERPLY_API_KEY`. Stored as `web.search.serply_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SERPLY_API_KEY`. Stored as `web.search.serply_api_key`. Sensitive.",
					},
					"ddgs_backend": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `DDGS_BACKEND`. Stored as `web.search.ddgs_backend`.",
						MarkdownDescription: "Open WebUI setting `DDGS_BACKEND`. Stored as `web.search.ddgs_backend`.",
					},
					"tavily_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `TAVILY_API_KEY`. Stored as `web.search.tavily_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `TAVILY_API_KEY`. Stored as `web.search.tavily_api_key`. Sensitive.",
					},
					"searchapi_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SEARCHAPI_API_KEY`. Stored as `web.search.searchapi_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SEARCHAPI_API_KEY`. Stored as `web.search.searchapi_api_key`. Sensitive.",
					},
					"searchapi_engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SEARCHAPI_ENGINE`. Stored as `web.search.searchapi_engine`.",
						MarkdownDescription: "Open WebUI setting `SEARCHAPI_ENGINE`. Stored as `web.search.searchapi_engine`.",
					},
					"serpapi_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SERPAPI_API_KEY`. Stored as `web.search.serpapi_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SERPAPI_API_KEY`. Stored as `web.search.serpapi_api_key`. Sensitive.",
					},
					"serpapi_engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SERPAPI_ENGINE`. Stored as `web.search.serpapi_engine`.",
						MarkdownDescription: "Open WebUI setting `SERPAPI_ENGINE`. Stored as `web.search.serpapi_engine`.",
					},
					"jina_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `JINA_API_KEY`. Stored as `web.search.jina_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `JINA_API_KEY`. Stored as `web.search.jina_api_key`. Sensitive.",
					},
					"jina_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `JINA_API_BASE_URL`. Stored as `web.search.jina_api_base_url`.",
						MarkdownDescription: "Open WebUI setting `JINA_API_BASE_URL`. Stored as `web.search.jina_api_base_url`.",
					},
					"bing_search_v7_endpoint": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `BING_SEARCH_V7_ENDPOINT`. Stored as `web.search.bing_search_v7_endpoint`.",
						MarkdownDescription: "Open WebUI setting `BING_SEARCH_V7_ENDPOINT`. Stored as `web.search.bing_search_v7_endpoint`.",
					},
					"bing_search_v7_subscription_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `BING_SEARCH_V7_SUBSCRIPTION_KEY`. Stored as `web.search.bing_search_v7_subscription_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `BING_SEARCH_V7_SUBSCRIPTION_KEY`. Stored as `web.search.bing_search_v7_subscription_key`. Sensitive.",
					},
					"exa_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `EXA_API_KEY`. Stored as `web.search.exa_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `EXA_API_KEY`. Stored as `web.search.exa_api_key`. Sensitive.",
					},
					"perplexity_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `PERPLEXITY_API_KEY`. Stored as `web.search.perplexity_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `PERPLEXITY_API_KEY`. Stored as `web.search.perplexity_api_key`. Sensitive.",
					},
					"perplexity_model": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `PERPLEXITY_MODEL`. Stored as `web.search.perplexity_model`.",
						MarkdownDescription: "Open WebUI setting `PERPLEXITY_MODEL`. Stored as `web.search.perplexity_model`.",
					},
					"perplexity_search_context_usage": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `PERPLEXITY_SEARCH_CONTEXT_USAGE`. Stored as `web.search.perplexity_search_context_usage`.",
						MarkdownDescription: "Open WebUI setting `PERPLEXITY_SEARCH_CONTEXT_USAGE`. Stored as `web.search.perplexity_search_context_usage`.",
					},
					"perplexity_search_api_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `PERPLEXITY_SEARCH_API_URL`. Stored as `web.search.perplexity_search_api_url`.",
						MarkdownDescription: "Open WebUI setting `PERPLEXITY_SEARCH_API_URL`. Stored as `web.search.perplexity_search_api_url`.",
					},
					"microsoft_web_iq_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MICROSOFT_WEB_IQ_API_BASE_URL`. Stored as `web.search.microsoft_web_iq_api_base_url`.",
						MarkdownDescription: "Open WebUI setting `MICROSOFT_WEB_IQ_API_BASE_URL`. Stored as `web.search.microsoft_web_iq_api_base_url`.",
					},
					"microsoft_web_iq_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `MICROSOFT_WEB_IQ_API_KEY`. Stored as `web.search.microsoft_web_iq_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `MICROSOFT_WEB_IQ_API_KEY`. Stored as `web.search.microsoft_web_iq_api_key`. Sensitive.",
					},
					"microsoft_web_iq_language": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MICROSOFT_WEB_IQ_LANGUAGE`. Stored as `web.search.microsoft_web_iq_language`.",
						MarkdownDescription: "Open WebUI setting `MICROSOFT_WEB_IQ_LANGUAGE`. Stored as `web.search.microsoft_web_iq_language`.",
					},
					"sougou_api_sid": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `SOUGOU_API_SID`. Stored as `web.search.sougou_api_sid`.",
						MarkdownDescription: "Open WebUI setting `SOUGOU_API_SID`. Stored as `web.search.sougou_api_sid`.",
					},
					"sougou_api_sk": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `SOUGOU_API_SK`. Stored as `web.search.sougou_api_sk`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `SOUGOU_API_SK`. Stored as `web.search.sougou_api_sk`. Sensitive.",
					},
					"web_loader_engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_LOADER_ENGINE`. Stored as `web.loader.engine`.",
						MarkdownDescription: "Open WebUI setting `WEB_LOADER_ENGINE`. Stored as `web.loader.engine`.",
					},
					"web_loader_timeout": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `WEB_LOADER_TIMEOUT`. Stored as `web.loader.timeout`.",
						MarkdownDescription: "Open WebUI setting `WEB_LOADER_TIMEOUT`. Stored as `web.loader.timeout`.",
					},
					"enable_web_loader_ssl_verification": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `ENABLE_WEB_LOADER_SSL_VERIFICATION`. Stored as `web.loader.ssl_verification`.",
						MarkdownDescription: "Open WebUI setting `ENABLE_WEB_LOADER_SSL_VERIFICATION`. Stored as `web.loader.ssl_verification`.",
					},
					"playwright_ws_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `PLAYWRIGHT_WS_URL`. Stored as `web.loader.playwright_ws_url`.",
						MarkdownDescription: "Open WebUI setting `PLAYWRIGHT_WS_URL`. Stored as `web.loader.playwright_ws_url`.",
					},
					"playwright_timeout": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `PLAYWRIGHT_TIMEOUT`. Stored as `web.loader.playwright_timeout`.",
						MarkdownDescription: "Open WebUI setting `PLAYWRIGHT_TIMEOUT`. Stored as `web.loader.playwright_timeout`.",
					},
					"firecrawl_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `FIRECRAWL_API_KEY`. Stored as `web.loader.firecrawl_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `FIRECRAWL_API_KEY`. Stored as `web.loader.firecrawl_api_key`. Sensitive.",
					},
					"firecrawl_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `FIRECRAWL_API_BASE_URL`. Stored as `web.loader.firecrawl_api_url`.",
						MarkdownDescription: "Open WebUI setting `FIRECRAWL_API_BASE_URL`. Stored as `web.loader.firecrawl_api_url`.",
					},
					"firecrawl_timeout": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `FIRECRAWL_TIMEOUT`. Stored as `web.loader.firecrawl_timeout`.",
						MarkdownDescription: "Open WebUI setting `FIRECRAWL_TIMEOUT`. Stored as `web.loader.firecrawl_timeout`.",
					},
					"tavily_extract_depth": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `TAVILY_EXTRACT_DEPTH`. Stored as `web.search.tavily_extract_depth`.",
						MarkdownDescription: "Open WebUI setting `TAVILY_EXTRACT_DEPTH`. Stored as `web.search.tavily_extract_depth`.",
					},
					"external_web_search_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `EXTERNAL_WEB_SEARCH_URL`. Stored as `web.search.external_web_search_url`.",
						MarkdownDescription: "Open WebUI setting `EXTERNAL_WEB_SEARCH_URL`. Stored as `web.search.external_web_search_url`.",
					},
					"external_web_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `EXTERNAL_WEB_SEARCH_API_KEY`. Stored as `web.search.external_web_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `EXTERNAL_WEB_SEARCH_API_KEY`. Stored as `web.search.external_web_search_api_key`. Sensitive.",
					},
					"external_web_loader_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `EXTERNAL_WEB_LOADER_URL`. Stored as `web.loader.external_web_loader_url`.",
						MarkdownDescription: "Open WebUI setting `EXTERNAL_WEB_LOADER_URL`. Stored as `web.loader.external_web_loader_url`.",
					},
					"external_web_loader_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `EXTERNAL_WEB_LOADER_API_KEY`. Stored as `web.loader.external_web_loader_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `EXTERNAL_WEB_LOADER_API_KEY`. Stored as `web.loader.external_web_loader_api_key`. Sensitive.",
					},
					"youtube_loader_language": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YOUTUBE_LOADER_LANGUAGE`. Stored as `rag.youtube_loader_language`.",
						MarkdownDescription: "Open WebUI setting `YOUTUBE_LOADER_LANGUAGE`. Stored as `rag.youtube_loader_language`.",
					},
					"youtube_loader_proxy_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YOUTUBE_LOADER_PROXY_URL`. Stored as `rag.youtube_loader_proxy_url`.",
						MarkdownDescription: "Open WebUI setting `YOUTUBE_LOADER_PROXY_URL`. Stored as `rag.youtube_loader_proxy_url`.",
					},
					"yandex_web_search_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YANDEX_WEB_SEARCH_URL`. Stored as `web.search.yandex_web_search_url`.",
						MarkdownDescription: "Open WebUI setting `YANDEX_WEB_SEARCH_URL`. Stored as `web.search.yandex_web_search_url`.",
					},
					"yandex_web_search_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `YANDEX_WEB_SEARCH_API_KEY`. Stored as `web.search.yandex_web_search_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `YANDEX_WEB_SEARCH_API_KEY`. Stored as `web.search.yandex_web_search_api_key`. Sensitive.",
					},
					"yandex_web_search_config": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `YANDEX_WEB_SEARCH_CONFIG`. Stored as `web.search.yandex_web_search_config`.",
						MarkdownDescription: "Open WebUI setting `YANDEX_WEB_SEARCH_CONFIG`. Stored as `web.search.yandex_web_search_config`.",
					},
					"youcom_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `YOUCOM_API_KEY`. Stored as `web.search.youcom_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `YOUCOM_API_KEY`. Stored as `web.search.youcom_api_key`. Sensitive.",
					},
					"linkup_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `LINKUP_API_KEY`. Stored as `web.search.linkup_api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `LINKUP_API_KEY`. Stored as `web.search.linkup_api_key`. Sensitive.",
					},
					"linkup_search_params": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "JSON object of extra parameters passed to Linkup. Stored as `web.search.linkup_search_params`.",
						MarkdownDescription: "JSON object of extra parameters passed to Linkup. Stored as `web.search.linkup_search_params`.",
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *ragConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the planned the RAG config.
func (r *ragConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG config.")
		return
	}

	var plan ragConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyRAGConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the the RAG config from Open WebUI.
func (r *ragConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG config.")
		return
	}

	var recorded ragConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := readRAGConfig(ctx, r.client, recorded, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the planned the RAG config.
func (r *ragConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG config.")
		return
	}

	var plan ragConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyRAGConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves Open WebUI's settings alone.
// There is no route that restores a default, and a config surface outlives the
// Terraform resource that describes it.
func (r *ragConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *ragConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyRAGConfig overlays the planned attributes on the current configuration,
// writes the whole form, and reads the result back. The update route's own
// response omits DATALAB_MARKER_FORMAT_LINES, DDGS_BACKEND,
// ENABLE_RAG_HYBRID_SEARCH_ENRICHED_TEXTS, MINERU_FILE_EXTENSIONS, and
// RAG_RERANKING_BATCH_SIZE, all of which the GET returns.
func applyRAGConfig(ctx context.Context, apiClient *client.Client, plan ragConfigModel, diags *diag.Diagnostics) ragConfigModel {
	current, err := apiClient.GetRAGConfig(ctx)
	if err != nil {
		diags.AddError("Read RAG config failed", err.Error())
		return ragConfigModel{}
	}

	form := expandRAGConfig(ctx, plan, current, diags)
	if diags.HasError() {
		return ragConfigModel{}
	}

	updated, err := apiClient.SetRAGConfig(ctx, form)
	if err != nil {
		diags.AddError("Update RAG config failed", err.Error())
		return ragConfigModel{}
	}

	return flattenRAGConfig(ctx, plan, updated, diags)
}

// readRAGConfig refreshes the recorded state from Open WebUI.
func readRAGConfig(ctx context.Context, apiClient *client.Client, recorded ragConfigModel, diags *diag.Diagnostics) ragConfigModel {
	current, err := apiClient.GetRAGConfig(ctx)
	if err != nil {
		diags.AddError("Read RAG config failed", err.Error())
		return ragConfigModel{}
	}

	return flattenRAGConfig(ctx, recorded, current, diags)
}

// expandRAGConfig builds the request form, taking each field from the plan when
// the plan carries a value and from the current configuration otherwise. A
// configuration with no web block leaves the block out of the request, which
// keeps every web key as it stands.
func expandRAGConfig(ctx context.Context, plan ragConfigModel, current *client.RAGConfigForm, diags *diag.Diagnostics) client.RAGConfigForm {
	root := path.Empty()
	form := client.RAGConfigForm{}
	form.RAGTemplate = engineString(plan.RAGTemplate, current.RAGTemplate)
	form.TopK = engineInt64(plan.TopK, current.TopK)
	form.BypassEmbeddingAndRetrieval = engineBool(plan.BypassEmbeddingAndRetrieval, current.BypassEmbeddingAndRetrieval)
	form.RAGFullContext = engineBool(plan.RAGFullContext, current.RAGFullContext)
	form.EnableRAGHybridSearch = engineBool(plan.EnableRAGHybridSearch, current.EnableRAGHybridSearch)
	form.EnableRAGHybridSearchEnrichedTexts = engineBool(plan.EnableRAGHybridSearchEnrichedTexts, current.EnableRAGHybridSearchEnrichedTexts)
	form.TopKReranker = engineInt64(plan.TopKReranker, current.TopKReranker)
	form.RelevanceThreshold = engineFloat64(plan.RelevanceThreshold, current.RelevanceThreshold)
	form.HybridBM25Weight = engineFloat64(plan.HybridBM25Weight, current.HybridBM25Weight)
	form.ContentExtractionEngine = engineString(plan.ContentExtractionEngine, current.ContentExtractionEngine)
	form.ContentExtractionSupportedMediaMimeTypes = engineStringList(ctx, plan.ContentExtractionSupportedMediaMimeTypes, current.ContentExtractionSupportedMediaMimeTypes, root.AtName("content_extraction_supported_media_mime_types"), diags)
	form.PDFExtractImages = engineBool(plan.PDFExtractImages, current.PDFExtractImages)
	form.PDFLoaderMode = engineString(plan.PDFLoaderMode, current.PDFLoaderMode)
	form.DatalabMarkerAPIKey = engineString(plan.DatalabMarkerAPIKey, current.DatalabMarkerAPIKey)
	form.DatalabMarkerAPIBaseURL = engineString(plan.DatalabMarkerAPIBaseURL, current.DatalabMarkerAPIBaseURL)
	form.DatalabMarkerAdditionalConfig = engineString(plan.DatalabMarkerAdditionalConfig, current.DatalabMarkerAdditionalConfig)
	form.DatalabMarkerSkipCache = engineBool(plan.DatalabMarkerSkipCache, current.DatalabMarkerSkipCache)
	form.DatalabMarkerForceOCR = engineBool(plan.DatalabMarkerForceOCR, current.DatalabMarkerForceOCR)
	form.DatalabMarkerPaginate = engineBool(plan.DatalabMarkerPaginate, current.DatalabMarkerPaginate)
	form.DatalabMarkerStripExistingOCR = engineBool(plan.DatalabMarkerStripExistingOCR, current.DatalabMarkerStripExistingOCR)
	form.DatalabMarkerDisableImageExtraction = engineBool(plan.DatalabMarkerDisableImageExtraction, current.DatalabMarkerDisableImageExtraction)
	form.DatalabMarkerFormatLines = engineBool(plan.DatalabMarkerFormatLines, current.DatalabMarkerFormatLines)
	form.DatalabMarkerUseLLM = engineBool(plan.DatalabMarkerUseLLM, current.DatalabMarkerUseLLM)
	form.DatalabMarkerOutputFormat = engineString(plan.DatalabMarkerOutputFormat, current.DatalabMarkerOutputFormat)
	form.ExternalDocumentLoaderURL = engineString(plan.ExternalDocumentLoaderURL, current.ExternalDocumentLoaderURL)
	form.ExternalDocumentLoaderAPIKey = engineString(plan.ExternalDocumentLoaderAPIKey, current.ExternalDocumentLoaderAPIKey)
	form.ExternalDocumentLoaderHeaders = engineJSON(plan.ExternalDocumentLoaderHeaders, current.ExternalDocumentLoaderHeaders, root.AtName("external_document_loader_headers"), diags)
	form.TikaServerURL = engineString(plan.TikaServerURL, current.TikaServerURL)
	form.DoclingServerURL = engineString(plan.DoclingServerURL, current.DoclingServerURL)
	form.DoclingAPIKey = engineString(plan.DoclingAPIKey, current.DoclingAPIKey)
	form.DoclingParams = engineJSON(plan.DoclingParams, current.DoclingParams, root.AtName("docling_params"), diags)
	form.DocumentIntelligenceEndpoint = engineString(plan.DocumentIntelligenceEndpoint, current.DocumentIntelligenceEndpoint)
	form.DocumentIntelligenceKey = engineString(plan.DocumentIntelligenceKey, current.DocumentIntelligenceKey)
	form.DocumentIntelligenceModel = engineString(plan.DocumentIntelligenceModel, current.DocumentIntelligenceModel)
	form.MistralOCRAPIBaseURL = engineString(plan.MistralOCRAPIBaseURL, current.MistralOCRAPIBaseURL)
	form.MistralOCRAPIKey = engineString(plan.MistralOCRAPIKey, current.MistralOCRAPIKey)
	form.MistralOCRUseBase64 = engineBool(plan.MistralOCRUseBase64, current.MistralOCRUseBase64)
	form.PaddleocrVLBaseURL = engineString(plan.PaddleocrVLBaseURL, current.PaddleocrVLBaseURL)
	form.PaddleocrVLToken = engineString(plan.PaddleocrVLToken, current.PaddleocrVLToken)
	form.MineruAPIMode = engineString(plan.MineruAPIMode, current.MineruAPIMode)
	form.MineruAPIURL = engineString(plan.MineruAPIURL, current.MineruAPIURL)
	form.MineruAPIKey = engineString(plan.MineruAPIKey, current.MineruAPIKey)
	form.MineruAPITimeout = engineNumericInt(plan.MineruAPITimeout, current.MineruAPITimeout)
	form.MineruParams = engineJSON(plan.MineruParams, current.MineruParams, root.AtName("mineru_params"), diags)
	form.MineruFileExtensions = engineStringList(ctx, plan.MineruFileExtensions, current.MineruFileExtensions, root.AtName("mineru_file_extensions"), diags)
	form.RAGRerankingModel = engineString(plan.RAGRerankingModel, current.RAGRerankingModel)
	form.RAGRerankingEngine = engineString(plan.RAGRerankingEngine, current.RAGRerankingEngine)
	form.RAGRerankingBatchSize = engineInt64(plan.RAGRerankingBatchSize, current.RAGRerankingBatchSize)
	form.RAGExternalRerankerURL = engineString(plan.RAGExternalRerankerURL, current.RAGExternalRerankerURL)
	form.RAGExternalRerankerAPIKey = engineString(plan.RAGExternalRerankerAPIKey, current.RAGExternalRerankerAPIKey)
	form.RAGExternalRerankerTimeout = engineString(plan.RAGExternalRerankerTimeout, current.RAGExternalRerankerTimeout)
	form.TextSplitter = engineString(plan.TextSplitter, current.TextSplitter)
	form.RAGTokenizerModel = engineString(plan.RAGTokenizerModel, current.RAGTokenizerModel)
	form.EnableMarkdownHeaderTextSplitter = engineBool(plan.EnableMarkdownHeaderTextSplitter, current.EnableMarkdownHeaderTextSplitter)
	form.ChunkSize = engineInt64(plan.ChunkSize, current.ChunkSize)
	form.ChunkMinSizeTarget = engineInt64(plan.ChunkMinSizeTarget, current.ChunkMinSizeTarget)
	form.ChunkOverlap = engineInt64(plan.ChunkOverlap, current.ChunkOverlap)
	form.FileMaxSize = engineNumericString(plan.FileMaxSize, current.FileMaxSize)
	form.FileMaxCount = engineNumericString(plan.FileMaxCount, current.FileMaxCount)
	form.FileImageCompressionWidth = engineNumericString(plan.FileImageCompressionWidth, current.FileImageCompressionWidth)
	form.FileImageCompressionHeight = engineNumericString(plan.FileImageCompressionHeight, current.FileImageCompressionHeight)
	form.AllowedFileExtensions = engineStringList(ctx, plan.AllowedFileExtensions, current.AllowedFileExtensions, root.AtName("allowed_file_extensions"), diags)
	form.EnableGoogleDriveIntegration = engineBool(plan.EnableGoogleDriveIntegration, current.EnableGoogleDriveIntegration)
	form.EnableOneDriveIntegration = engineBool(plan.EnableOneDriveIntegration, current.EnableOneDriveIntegration)

	if plan.Web == nil {
		return form
	}

	webRoot := path.Root("web")
	currentWeb := current.Web
	if currentWeb == nil {
		currentWeb = &client.RAGWebConfigForm{}
	}

	web := &client.RAGWebConfigForm{}
	web.EnableWebSearch = engineBool(plan.Web.EnableWebSearch, currentWeb.EnableWebSearch)
	web.EnableWebSearchConfirmation = engineBool(plan.Web.EnableWebSearchConfirmation, currentWeb.EnableWebSearchConfirmation)
	web.WebSearchConfirmationContent = engineString(plan.Web.WebSearchConfirmationContent, currentWeb.WebSearchConfirmationContent)
	web.WebSearchEngine = engineString(plan.Web.WebSearchEngine, currentWeb.WebSearchEngine)
	web.WebSearchTrustEnv = engineBool(plan.Web.WebSearchTrustEnv, currentWeb.WebSearchTrustEnv)
	web.WebSearchResultCount = engineInt64(plan.Web.WebSearchResultCount, currentWeb.WebSearchResultCount)
	web.WebSearchConcurrentRequests = engineInt64(plan.Web.WebSearchConcurrentRequests, currentWeb.WebSearchConcurrentRequests)
	web.WebSearchDomainFilterList = engineStringList(ctx, plan.Web.WebSearchDomainFilterList, currentWeb.WebSearchDomainFilterList, webRoot.AtName("web_search_domain_filter_list"), diags)
	web.WebFetchMaxContentLength = engineInt64(plan.Web.WebFetchMaxContentLength, currentWeb.WebFetchMaxContentLength)
	web.WebLoaderConcurrentRequests = engineInt64(plan.Web.WebLoaderConcurrentRequests, currentWeb.WebLoaderConcurrentRequests)
	web.BypassWebSearchEmbeddingAndRetrieval = engineBool(plan.Web.BypassWebSearchEmbeddingAndRetrieval, currentWeb.BypassWebSearchEmbeddingAndRetrieval)
	web.BypassWebSearchWebLoader = engineBool(plan.Web.BypassWebSearchWebLoader, currentWeb.BypassWebSearchWebLoader)
	web.OllamaCloudWebSearchAPIKey = engineString(plan.Web.OllamaCloudWebSearchAPIKey, currentWeb.OllamaCloudWebSearchAPIKey)
	web.SearxngQueryURL = engineString(plan.Web.SearxngQueryURL, currentWeb.SearxngQueryURL)
	web.SearxngLanguage = engineString(plan.Web.SearxngLanguage, currentWeb.SearxngLanguage)
	web.OpenserpBaseURL = engineString(plan.Web.OpenserpBaseURL, currentWeb.OpenserpBaseURL)
	web.YacyQueryURL = engineString(plan.Web.YacyQueryURL, currentWeb.YacyQueryURL)
	web.YacyUsername = engineString(plan.Web.YacyUsername, currentWeb.YacyUsername)
	web.YacyPassword = engineString(plan.Web.YacyPassword, currentWeb.YacyPassword)
	web.GooglePSEAPIKey = engineString(plan.Web.GooglePSEAPIKey, currentWeb.GooglePSEAPIKey)
	web.GooglePSEEngineID = engineString(plan.Web.GooglePSEEngineID, currentWeb.GooglePSEEngineID)
	web.BraveSearchAPIKey = engineString(plan.Web.BraveSearchAPIKey, currentWeb.BraveSearchAPIKey)
	web.BraveSearchContextTokens = engineInt64(plan.Web.BraveSearchContextTokens, currentWeb.BraveSearchContextTokens)
	web.KagiSearchAPIKey = engineString(plan.Web.KagiSearchAPIKey, currentWeb.KagiSearchAPIKey)
	web.MojeekSearchAPIKey = engineString(plan.Web.MojeekSearchAPIKey, currentWeb.MojeekSearchAPIKey)
	web.BochaSearchAPIKey = engineString(plan.Web.BochaSearchAPIKey, currentWeb.BochaSearchAPIKey)
	web.SerpstackAPIKey = engineString(plan.Web.SerpstackAPIKey, currentWeb.SerpstackAPIKey)
	web.SerpstackHTTPS = engineBool(plan.Web.SerpstackHTTPS, currentWeb.SerpstackHTTPS)
	web.SerperAPIKey = engineString(plan.Web.SerperAPIKey, currentWeb.SerperAPIKey)
	web.SerphouseAPIKey = engineString(plan.Web.SerphouseAPIKey, currentWeb.SerphouseAPIKey)
	web.SerphouseDomain = engineString(plan.Web.SerphouseDomain, currentWeb.SerphouseDomain)
	web.SerplyAPIKey = engineString(plan.Web.SerplyAPIKey, currentWeb.SerplyAPIKey)
	web.DdgsBackend = engineString(plan.Web.DdgsBackend, currentWeb.DdgsBackend)
	web.TavilyAPIKey = engineString(plan.Web.TavilyAPIKey, currentWeb.TavilyAPIKey)
	web.SearchapiAPIKey = engineString(plan.Web.SearchapiAPIKey, currentWeb.SearchapiAPIKey)
	web.SearchapiEngine = engineString(plan.Web.SearchapiEngine, currentWeb.SearchapiEngine)
	web.SerpapiAPIKey = engineString(plan.Web.SerpapiAPIKey, currentWeb.SerpapiAPIKey)
	web.SerpapiEngine = engineString(plan.Web.SerpapiEngine, currentWeb.SerpapiEngine)
	web.JinaAPIKey = engineString(plan.Web.JinaAPIKey, currentWeb.JinaAPIKey)
	web.JinaAPIBaseURL = engineString(plan.Web.JinaAPIBaseURL, currentWeb.JinaAPIBaseURL)
	web.BingSearchV7Endpoint = engineString(plan.Web.BingSearchV7Endpoint, currentWeb.BingSearchV7Endpoint)
	web.BingSearchV7SubscriptionKey = engineString(plan.Web.BingSearchV7SubscriptionKey, currentWeb.BingSearchV7SubscriptionKey)
	web.ExaAPIKey = engineString(plan.Web.ExaAPIKey, currentWeb.ExaAPIKey)
	web.PerplexityAPIKey = engineString(plan.Web.PerplexityAPIKey, currentWeb.PerplexityAPIKey)
	web.PerplexityModel = engineString(plan.Web.PerplexityModel, currentWeb.PerplexityModel)
	web.PerplexitySearchContextUsage = engineString(plan.Web.PerplexitySearchContextUsage, currentWeb.PerplexitySearchContextUsage)
	web.PerplexitySearchAPIURL = engineString(plan.Web.PerplexitySearchAPIURL, currentWeb.PerplexitySearchAPIURL)
	web.MicrosoftWebIQAPIBaseURL = engineString(plan.Web.MicrosoftWebIQAPIBaseURL, currentWeb.MicrosoftWebIQAPIBaseURL)
	web.MicrosoftWebIQAPIKey = engineString(plan.Web.MicrosoftWebIQAPIKey, currentWeb.MicrosoftWebIQAPIKey)
	web.MicrosoftWebIQLanguage = engineString(plan.Web.MicrosoftWebIQLanguage, currentWeb.MicrosoftWebIQLanguage)
	web.SougouAPISID = engineString(plan.Web.SougouAPISID, currentWeb.SougouAPISID)
	web.SougouAPISK = engineString(plan.Web.SougouAPISK, currentWeb.SougouAPISK)
	web.WebLoaderEngine = engineString(plan.Web.WebLoaderEngine, currentWeb.WebLoaderEngine)
	web.WebLoaderTimeout = engineString(plan.Web.WebLoaderTimeout, currentWeb.WebLoaderTimeout)
	web.EnableWebLoaderSSLVerification = engineBool(plan.Web.EnableWebLoaderSSLVerification, currentWeb.EnableWebLoaderSSLVerification)
	web.PlaywrightWSURL = engineString(plan.Web.PlaywrightWSURL, currentWeb.PlaywrightWSURL)
	web.PlaywrightTimeout = engineInt64(plan.Web.PlaywrightTimeout, currentWeb.PlaywrightTimeout)
	web.FirecrawlAPIKey = engineString(plan.Web.FirecrawlAPIKey, currentWeb.FirecrawlAPIKey)
	web.FirecrawlAPIBaseURL = engineString(plan.Web.FirecrawlAPIBaseURL, currentWeb.FirecrawlAPIBaseURL)
	web.FirecrawlTimeout = engineString(plan.Web.FirecrawlTimeout, currentWeb.FirecrawlTimeout)
	web.TavilyExtractDepth = engineString(plan.Web.TavilyExtractDepth, currentWeb.TavilyExtractDepth)
	web.ExternalWebSearchURL = engineString(plan.Web.ExternalWebSearchURL, currentWeb.ExternalWebSearchURL)
	web.ExternalWebSearchAPIKey = engineString(plan.Web.ExternalWebSearchAPIKey, currentWeb.ExternalWebSearchAPIKey)
	web.ExternalWebLoaderURL = engineString(plan.Web.ExternalWebLoaderURL, currentWeb.ExternalWebLoaderURL)
	web.ExternalWebLoaderAPIKey = engineString(plan.Web.ExternalWebLoaderAPIKey, currentWeb.ExternalWebLoaderAPIKey)
	web.YoutubeLoaderLanguage = engineStringList(ctx, plan.Web.YoutubeLoaderLanguage, currentWeb.YoutubeLoaderLanguage, webRoot.AtName("youtube_loader_language"), diags)
	web.YoutubeLoaderProxyURL = engineString(plan.Web.YoutubeLoaderProxyURL, currentWeb.YoutubeLoaderProxyURL)
	web.YandexWebSearchURL = engineString(plan.Web.YandexWebSearchURL, currentWeb.YandexWebSearchURL)
	web.YandexWebSearchAPIKey = engineString(plan.Web.YandexWebSearchAPIKey, currentWeb.YandexWebSearchAPIKey)
	web.YandexWebSearchConfig = engineString(plan.Web.YandexWebSearchConfig, currentWeb.YandexWebSearchConfig)
	web.YoucomAPIKey = engineString(plan.Web.YoucomAPIKey, currentWeb.YoucomAPIKey)
	web.LinkupAPIKey = engineString(plan.Web.LinkupAPIKey, currentWeb.LinkupAPIKey)
	web.LinkupSearchParams = engineJSON(plan.Web.LinkupSearchParams, currentWeb.LinkupSearchParams, webRoot.AtName("linkup_search_params"), diags)

	form.Web = web

	return form
}

// flattenRAGConfig records the configuration Open WebUI returned. The web block
// stays out of state until a configuration names it, because a resource that
// does not manage the block must not report its values as managed.
func flattenRAGConfig(ctx context.Context, prior ragConfigModel, remote *client.RAGConfigForm, diags *diag.Diagnostics) ragConfigModel {
	state := ragConfigModel{ID: types.StringValue("rag")}
	state.RAGTemplate = stringValueOrNull(remote.RAGTemplate)
	state.TopK = int64ValueOrNull(remote.TopK)
	state.BypassEmbeddingAndRetrieval = engineBoolValue(remote.BypassEmbeddingAndRetrieval)
	state.RAGFullContext = engineBoolValue(remote.RAGFullContext)
	state.EnableRAGHybridSearch = engineBoolValue(remote.EnableRAGHybridSearch)
	state.EnableRAGHybridSearchEnrichedTexts = engineBoolValue(remote.EnableRAGHybridSearchEnrichedTexts)
	state.TopKReranker = int64ValueOrNull(remote.TopKReranker)
	state.RelevanceThreshold = engineFloat64Value(remote.RelevanceThreshold)
	state.HybridBM25Weight = engineFloat64Value(remote.HybridBM25Weight)
	state.ContentExtractionEngine = stringValueOrNull(remote.ContentExtractionEngine)
	state.ContentExtractionSupportedMediaMimeTypes = engineStringListValue(ctx, remote.ContentExtractionSupportedMediaMimeTypes, diags)
	state.PDFExtractImages = engineBoolValue(remote.PDFExtractImages)
	state.PDFLoaderMode = stringValueOrNull(remote.PDFLoaderMode)
	state.DatalabMarkerAPIKey = stringValueOrNull(remote.DatalabMarkerAPIKey)
	state.DatalabMarkerAPIBaseURL = stringValueOrNull(remote.DatalabMarkerAPIBaseURL)
	state.DatalabMarkerAdditionalConfig = stringValueOrNull(remote.DatalabMarkerAdditionalConfig)
	state.DatalabMarkerSkipCache = engineBoolValue(remote.DatalabMarkerSkipCache)
	state.DatalabMarkerForceOCR = engineBoolValue(remote.DatalabMarkerForceOCR)
	state.DatalabMarkerPaginate = engineBoolValue(remote.DatalabMarkerPaginate)
	state.DatalabMarkerStripExistingOCR = engineBoolValue(remote.DatalabMarkerStripExistingOCR)
	state.DatalabMarkerDisableImageExtraction = engineBoolValue(remote.DatalabMarkerDisableImageExtraction)
	state.DatalabMarkerFormatLines = engineBoolValue(remote.DatalabMarkerFormatLines)
	state.DatalabMarkerUseLLM = engineBoolValue(remote.DatalabMarkerUseLLM)
	state.DatalabMarkerOutputFormat = stringValueOrNull(remote.DatalabMarkerOutputFormat)
	state.ExternalDocumentLoaderURL = stringValueOrNull(remote.ExternalDocumentLoaderURL)
	state.ExternalDocumentLoaderAPIKey = stringValueOrNull(remote.ExternalDocumentLoaderAPIKey)
	state.ExternalDocumentLoaderHeaders = engineJSONValue(prior.ExternalDocumentLoaderHeaders, remote.ExternalDocumentLoaderHeaders, "external_document_loader_headers", diags)
	state.TikaServerURL = stringValueOrNull(remote.TikaServerURL)
	state.DoclingServerURL = stringValueOrNull(remote.DoclingServerURL)
	state.DoclingAPIKey = stringValueOrNull(remote.DoclingAPIKey)
	state.DoclingParams = engineJSONValue(prior.DoclingParams, remote.DoclingParams, "docling_params", diags)
	state.DocumentIntelligenceEndpoint = stringValueOrNull(remote.DocumentIntelligenceEndpoint)
	state.DocumentIntelligenceKey = stringValueOrNull(remote.DocumentIntelligenceKey)
	state.DocumentIntelligenceModel = stringValueOrNull(remote.DocumentIntelligenceModel)
	state.MistralOCRAPIBaseURL = stringValueOrNull(remote.MistralOCRAPIBaseURL)
	state.MistralOCRAPIKey = stringValueOrNull(remote.MistralOCRAPIKey)
	state.MistralOCRUseBase64 = engineBoolValue(remote.MistralOCRUseBase64)
	state.PaddleocrVLBaseURL = stringValueOrNull(remote.PaddleocrVLBaseURL)
	state.PaddleocrVLToken = stringValueOrNull(remote.PaddleocrVLToken)
	state.MineruAPIMode = stringValueOrNull(remote.MineruAPIMode)
	state.MineruAPIURL = stringValueOrNull(remote.MineruAPIURL)
	state.MineruAPIKey = stringValueOrNull(remote.MineruAPIKey)
	state.MineruAPITimeout = engineNumericIntValue(remote.MineruAPITimeout)
	state.MineruParams = engineJSONValue(prior.MineruParams, remote.MineruParams, "mineru_params", diags)
	state.MineruFileExtensions = engineStringListValue(ctx, remote.MineruFileExtensions, diags)
	state.RAGRerankingModel = stringValueOrNull(remote.RAGRerankingModel)
	state.RAGRerankingEngine = stringValueOrNull(remote.RAGRerankingEngine)
	state.RAGRerankingBatchSize = int64ValueOrNull(remote.RAGRerankingBatchSize)
	state.RAGExternalRerankerURL = stringValueOrNull(remote.RAGExternalRerankerURL)
	state.RAGExternalRerankerAPIKey = stringValueOrNull(remote.RAGExternalRerankerAPIKey)
	state.RAGExternalRerankerTimeout = stringValueOrNull(remote.RAGExternalRerankerTimeout)
	state.TextSplitter = stringValueOrNull(remote.TextSplitter)
	state.RAGTokenizerModel = stringValueOrNull(remote.RAGTokenizerModel)
	state.EnableMarkdownHeaderTextSplitter = engineBoolValue(remote.EnableMarkdownHeaderTextSplitter)
	state.ChunkSize = int64ValueOrNull(remote.ChunkSize)
	state.ChunkMinSizeTarget = int64ValueOrNull(remote.ChunkMinSizeTarget)
	state.ChunkOverlap = int64ValueOrNull(remote.ChunkOverlap)
	state.FileMaxSize = engineNumericStringValue(prior.FileMaxSize, remote.FileMaxSize)
	state.FileMaxCount = engineNumericStringValue(prior.FileMaxCount, remote.FileMaxCount)
	state.FileImageCompressionWidth = engineNumericStringValue(prior.FileImageCompressionWidth, remote.FileImageCompressionWidth)
	state.FileImageCompressionHeight = engineNumericStringValue(prior.FileImageCompressionHeight, remote.FileImageCompressionHeight)
	state.AllowedFileExtensions = engineStringListValue(ctx, remote.AllowedFileExtensions, diags)
	state.EnableGoogleDriveIntegration = engineBoolValue(remote.EnableGoogleDriveIntegration)
	state.EnableOneDriveIntegration = engineBoolValue(remote.EnableOneDriveIntegration)

	if prior.Web == nil || remote.Web == nil {
		return state
	}

	priorWeb := prior.Web
	web := &ragWebConfigModel{}
	web.EnableWebSearch = engineBoolValue(remote.Web.EnableWebSearch)
	web.EnableWebSearchConfirmation = engineBoolValue(remote.Web.EnableWebSearchConfirmation)
	web.WebSearchConfirmationContent = stringValueOrNull(remote.Web.WebSearchConfirmationContent)
	web.WebSearchEngine = stringValueOrNull(remote.Web.WebSearchEngine)
	web.WebSearchTrustEnv = engineBoolValue(remote.Web.WebSearchTrustEnv)
	web.WebSearchResultCount = int64ValueOrNull(remote.Web.WebSearchResultCount)
	web.WebSearchConcurrentRequests = int64ValueOrNull(remote.Web.WebSearchConcurrentRequests)
	web.WebSearchDomainFilterList = engineStringListValue(ctx, remote.Web.WebSearchDomainFilterList, diags)
	web.WebFetchMaxContentLength = int64ValueOrNull(remote.Web.WebFetchMaxContentLength)
	web.WebLoaderConcurrentRequests = int64ValueOrNull(remote.Web.WebLoaderConcurrentRequests)
	web.BypassWebSearchEmbeddingAndRetrieval = engineBoolValue(remote.Web.BypassWebSearchEmbeddingAndRetrieval)
	web.BypassWebSearchWebLoader = engineBoolValue(remote.Web.BypassWebSearchWebLoader)
	web.OllamaCloudWebSearchAPIKey = stringValueOrNull(remote.Web.OllamaCloudWebSearchAPIKey)
	web.SearxngQueryURL = stringValueOrNull(remote.Web.SearxngQueryURL)
	web.SearxngLanguage = stringValueOrNull(remote.Web.SearxngLanguage)
	web.OpenserpBaseURL = stringValueOrNull(remote.Web.OpenserpBaseURL)
	web.YacyQueryURL = stringValueOrNull(remote.Web.YacyQueryURL)
	web.YacyUsername = stringValueOrNull(remote.Web.YacyUsername)
	web.YacyPassword = stringValueOrNull(remote.Web.YacyPassword)
	web.GooglePSEAPIKey = stringValueOrNull(remote.Web.GooglePSEAPIKey)
	web.GooglePSEEngineID = stringValueOrNull(remote.Web.GooglePSEEngineID)
	web.BraveSearchAPIKey = stringValueOrNull(remote.Web.BraveSearchAPIKey)
	web.BraveSearchContextTokens = int64ValueOrNull(remote.Web.BraveSearchContextTokens)
	web.KagiSearchAPIKey = stringValueOrNull(remote.Web.KagiSearchAPIKey)
	web.MojeekSearchAPIKey = stringValueOrNull(remote.Web.MojeekSearchAPIKey)
	web.BochaSearchAPIKey = stringValueOrNull(remote.Web.BochaSearchAPIKey)
	web.SerpstackAPIKey = stringValueOrNull(remote.Web.SerpstackAPIKey)
	web.SerpstackHTTPS = engineBoolValue(remote.Web.SerpstackHTTPS)
	web.SerperAPIKey = stringValueOrNull(remote.Web.SerperAPIKey)
	web.SerphouseAPIKey = stringValueOrNull(remote.Web.SerphouseAPIKey)
	web.SerphouseDomain = stringValueOrNull(remote.Web.SerphouseDomain)
	web.SerplyAPIKey = stringValueOrNull(remote.Web.SerplyAPIKey)
	web.DdgsBackend = stringValueOrNull(remote.Web.DdgsBackend)
	web.TavilyAPIKey = stringValueOrNull(remote.Web.TavilyAPIKey)
	web.SearchapiAPIKey = stringValueOrNull(remote.Web.SearchapiAPIKey)
	web.SearchapiEngine = stringValueOrNull(remote.Web.SearchapiEngine)
	web.SerpapiAPIKey = stringValueOrNull(remote.Web.SerpapiAPIKey)
	web.SerpapiEngine = stringValueOrNull(remote.Web.SerpapiEngine)
	web.JinaAPIKey = stringValueOrNull(remote.Web.JinaAPIKey)
	web.JinaAPIBaseURL = stringValueOrNull(remote.Web.JinaAPIBaseURL)
	web.BingSearchV7Endpoint = stringValueOrNull(remote.Web.BingSearchV7Endpoint)
	web.BingSearchV7SubscriptionKey = stringValueOrNull(remote.Web.BingSearchV7SubscriptionKey)
	web.ExaAPIKey = stringValueOrNull(remote.Web.ExaAPIKey)
	web.PerplexityAPIKey = stringValueOrNull(remote.Web.PerplexityAPIKey)
	web.PerplexityModel = stringValueOrNull(remote.Web.PerplexityModel)
	web.PerplexitySearchContextUsage = stringValueOrNull(remote.Web.PerplexitySearchContextUsage)
	web.PerplexitySearchAPIURL = stringValueOrNull(remote.Web.PerplexitySearchAPIURL)
	web.MicrosoftWebIQAPIBaseURL = stringValueOrNull(remote.Web.MicrosoftWebIQAPIBaseURL)
	web.MicrosoftWebIQAPIKey = stringValueOrNull(remote.Web.MicrosoftWebIQAPIKey)
	web.MicrosoftWebIQLanguage = stringValueOrNull(remote.Web.MicrosoftWebIQLanguage)
	web.SougouAPISID = stringValueOrNull(remote.Web.SougouAPISID)
	web.SougouAPISK = stringValueOrNull(remote.Web.SougouAPISK)
	web.WebLoaderEngine = stringValueOrNull(remote.Web.WebLoaderEngine)
	web.WebLoaderTimeout = stringValueOrNull(remote.Web.WebLoaderTimeout)
	web.EnableWebLoaderSSLVerification = engineBoolValue(remote.Web.EnableWebLoaderSSLVerification)
	web.PlaywrightWSURL = stringValueOrNull(remote.Web.PlaywrightWSURL)
	web.PlaywrightTimeout = int64ValueOrNull(remote.Web.PlaywrightTimeout)
	web.FirecrawlAPIKey = stringValueOrNull(remote.Web.FirecrawlAPIKey)
	web.FirecrawlAPIBaseURL = stringValueOrNull(remote.Web.FirecrawlAPIBaseURL)
	web.FirecrawlTimeout = stringValueOrNull(remote.Web.FirecrawlTimeout)
	web.TavilyExtractDepth = stringValueOrNull(remote.Web.TavilyExtractDepth)
	web.ExternalWebSearchURL = stringValueOrNull(remote.Web.ExternalWebSearchURL)
	web.ExternalWebSearchAPIKey = stringValueOrNull(remote.Web.ExternalWebSearchAPIKey)
	web.ExternalWebLoaderURL = stringValueOrNull(remote.Web.ExternalWebLoaderURL)
	web.ExternalWebLoaderAPIKey = stringValueOrNull(remote.Web.ExternalWebLoaderAPIKey)
	web.YoutubeLoaderLanguage = engineStringListValue(ctx, remote.Web.YoutubeLoaderLanguage, diags)
	web.YoutubeLoaderProxyURL = stringValueOrNull(remote.Web.YoutubeLoaderProxyURL)
	web.YandexWebSearchURL = stringValueOrNull(remote.Web.YandexWebSearchURL)
	web.YandexWebSearchAPIKey = stringValueOrNull(remote.Web.YandexWebSearchAPIKey)
	web.YandexWebSearchConfig = stringValueOrNull(remote.Web.YandexWebSearchConfig)
	web.YoucomAPIKey = stringValueOrNull(remote.Web.YoucomAPIKey)
	web.LinkupAPIKey = stringValueOrNull(remote.Web.LinkupAPIKey)
	web.LinkupSearchParams = engineJSONValue(priorWeb.LinkupSearchParams, remote.Web.LinkupSearchParams, "linkup_search_params", diags)

	state.Web = web

	return state
}

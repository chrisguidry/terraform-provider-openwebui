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
	registeredResources = append(registeredResources, NewRAGEmbeddingConfigResource)
}

var _ resource.Resource = &ragEmbeddingConfigResource{}
var _ resource.ResourceWithConfigure = &ragEmbeddingConfigResource{}
var _ resource.ResourceWithImportState = &ragEmbeddingConfigResource{}

// ragEmbeddingConfigResource manages the embedding engine and its credentials.
type ragEmbeddingConfigResource struct {
	client *client.Client
}

// ragEmbeddingConfigModel mirrors EmbeddingModelUpdateForm in
// backend/open_webui/routers/retrieval.py.
type ragEmbeddingConfigModel struct {
	ID                             types.String               `tfsdk:"id"`
	RAGEmbeddingEngine             types.String               `tfsdk:"rag_embedding_engine"`
	RAGEmbeddingModel              types.String               `tfsdk:"rag_embedding_model"`
	RAGEmbeddingBatchSize          types.Int64                `tfsdk:"rag_embedding_batch_size"`
	EnableAsyncEmbedding           types.Bool                 `tfsdk:"enable_async_embedding"`
	RAGEmbeddingConcurrentRequests types.Int64                `tfsdk:"rag_embedding_concurrent_requests"`
	OpenAIConfig                   *ragEngineConfigModel      `tfsdk:"openai_config"`
	OllamaConfig                   *ragEngineConfigModel      `tfsdk:"ollama_config"`
	AzureOpenAIConfig              *ragAzureEngineConfigModel `tfsdk:"azure_openai_config"`
}

// ragEngineConfigModel mirrors OpenAIConfigForm and OllamaConfigForm, which
// carry the same two fields.
type ragEngineConfigModel struct {
	URL types.String `tfsdk:"url"`
	Key types.String `tfsdk:"key"`
}

// ragAzureEngineConfigModel mirrors AzureOpenAIConfigForm, which adds the API version.
type ragAzureEngineConfigModel struct {
	URL     types.String `tfsdk:"url"`
	Key     types.String `tfsdk:"key"`
	Version types.String `tfsdk:"version"`
}

// NewRAGEmbeddingConfigResource constructs a new RAG embedding config resource.
func NewRAGEmbeddingConfigResource() resource.Resource {
	return &ragEmbeddingConfigResource{}
}

// Metadata sets the resource type name.
func (r *ragEmbeddingConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rag_embedding_config"
}

// Schema defines the RAG embedding config schema.
func (r *ragEmbeddingConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the embedding engine Open WebUI indexes and queries documents with.\n\n" +
			"`POST /api/v1/retrieval/embedding/update` assigns all five top-level keys from the form it receives, so this " +
			"resource reads the current configuration and overlays the attributes the plan names before it writes.\n\n" +
			"An engine block is written only when `rag_embedding_engine` is `ollama`, `openai`, or `azure_openai`. A block this " +
			"configuration leaves out keeps its stored values.\n\n" +
			"The update route loads the new embedding model into the Open WebUI process and unloads the previous one, so an " +
			"apply that switches engines does real work and can take a while. It is idempotent.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"rag_embedding_engine": schema.StringAttribute{
				Required:            true,
				Description:         "Embedding engine: `ollama`, `openai`, `azure_openai`, or the empty string for the built-in model. Stored as `rag.embedding_engine`.",
				MarkdownDescription: "Embedding engine: `ollama`, `openai`, `azure_openai`, or the empty string for the built-in model. Stored as `rag.embedding_engine`.",
			},
			"rag_embedding_model": schema.StringAttribute{
				Required:            true,
				Description:         "Embedding model name. Stored as `rag.embedding_model`.",
				MarkdownDescription: "Embedding model name. Stored as `rag.embedding_model`.",
			},
			"rag_embedding_batch_size": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Texts sent per embedding request. Stored as `rag.embedding_batch_size`.",
				MarkdownDescription: "Texts sent per embedding request. Stored as `rag.embedding_batch_size`.",
			},
			"enable_async_embedding": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether embedding requests run asynchronously. Stored as `rag.enable_async_embedding`.",
				MarkdownDescription: "Whether embedding requests run asynchronously. Stored as `rag.enable_async_embedding`.",
			},
			"rag_embedding_concurrent_requests": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Embedding requests in flight at once. Stored as `rag.embedding_concurrent_requests`.",
				MarkdownDescription: "Embedding requests in flight at once. Stored as `rag.embedding_concurrent_requests`.",
			},
			"openai_config": schema.SingleNestedAttribute{
				Optional:            true,
				Description:         "OpenAI embedding endpoint. Stored as `rag.openai.api_base_url` and `rag.openai.api_key`.",
				MarkdownDescription: "OpenAI embedding endpoint. Stored as `rag.openai.api_base_url` and `rag.openai.api_key`.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:            true,
						Description:         "Base URL of the OpenAI-compatible embedding API.",
						MarkdownDescription: "Base URL of the OpenAI-compatible embedding API.",
					},
					"key": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						Description:         "API key for the embedding endpoint. Sensitive.",
						MarkdownDescription: "API key for the embedding endpoint. Sensitive.",
					},
				},
			},
			"ollama_config": schema.SingleNestedAttribute{
				Optional:            true,
				Description:         "Ollama embedding endpoint. Stored as `rag.ollama.base_url` and `rag.ollama.api_key`.",
				MarkdownDescription: "Ollama embedding endpoint. Stored as `rag.ollama.base_url` and `rag.ollama.api_key`.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:            true,
						Description:         "Base URL of the Ollama server.",
						MarkdownDescription: "Base URL of the Ollama server.",
					},
					"key": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						Description:         "API key for the Ollama server. Sensitive.",
						MarkdownDescription: "API key for the Ollama server. Sensitive.",
					},
				},
			},
			"azure_openai_config": schema.SingleNestedAttribute{
				Optional:            true,
				Description:         "Azure OpenAI embedding endpoint. Stored as `rag.azure_openai.base_url`, `rag.azure_openai.api_key`, and `rag.azure_openai.api_version`.",
				MarkdownDescription: "Azure OpenAI embedding endpoint. Stored as `rag.azure_openai.base_url`, `rag.azure_openai.api_key`, and `rag.azure_openai.api_version`.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Required:            true,
						Description:         "Base URL of the Azure OpenAI deployment.",
						MarkdownDescription: "Base URL of the Azure OpenAI deployment.",
					},
					"key": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						Description:         "API key for the Azure OpenAI deployment. Sensitive.",
						MarkdownDescription: "API key for the Azure OpenAI deployment. Sensitive.",
					},
					"version": schema.StringAttribute{
						Required:            true,
						Description:         "Azure OpenAI API version.",
						MarkdownDescription: "Azure OpenAI API version.",
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *ragEmbeddingConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the planned embedding config.
func (r *ragEmbeddingConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG embedding config.")
		return
	}

	var plan ragEmbeddingConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyRAGEmbeddingConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the embedding config from Open WebUI.
func (r *ragEmbeddingConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG embedding config.")
		return
	}

	var recorded ragEmbeddingConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetRAGEmbeddingConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read RAG embedding config failed", err.Error())
		return
	}

	state := flattenRAGEmbeddingConfig(recorded, current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the planned embedding config.
func (r *ragEmbeddingConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the RAG embedding config.")
		return
	}

	var plan ragEmbeddingConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyRAGEmbeddingConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves the embedding engine alone.
// There is no route that restores a default, and a config surface outlives the
// Terraform resource that describes it.
func (r *ragEmbeddingConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *ragEmbeddingConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyRAGEmbeddingConfig overlays the planned attributes on the current
// configuration and writes the whole form. The route assigns all five top-level
// keys from the form, so a request that omits one replaces it with a Pydantic
// default: 1 texts per batch, asynchronous embedding on, and no concurrency.
func applyRAGEmbeddingConfig(ctx context.Context, apiClient *client.Client, plan ragEmbeddingConfigModel, diags *diag.Diagnostics) ragEmbeddingConfigModel {
	current, err := apiClient.GetRAGEmbeddingConfig(ctx)
	if err != nil {
		diags.AddError("Read RAG embedding config failed", err.Error())
		return ragEmbeddingConfigModel{}
	}

	form := client.RAGEmbeddingConfigForm{
		RAGEmbeddingEngine:             engineString(plan.RAGEmbeddingEngine, current.RAGEmbeddingEngine),
		RAGEmbeddingModel:              engineString(plan.RAGEmbeddingModel, current.RAGEmbeddingModel),
		RAGEmbeddingBatchSize:          engineInt64(plan.RAGEmbeddingBatchSize, current.RAGEmbeddingBatchSize),
		EnableAsyncEmbedding:           engineBool(plan.EnableAsyncEmbedding, current.EnableAsyncEmbedding),
		RAGEmbeddingConcurrentRequests: engineInt64(plan.RAGEmbeddingConcurrentRequests, current.RAGEmbeddingConcurrentRequests),
	}

	if plan.OpenAIConfig != nil {
		form.OpenAIConfig = &client.RAGEngineConfigForm{
			URL: stringPtr(plan.OpenAIConfig.URL),
			Key: stringPtr(plan.OpenAIConfig.Key),
		}
	}

	if plan.OllamaConfig != nil {
		form.OllamaConfig = &client.RAGEngineConfigForm{
			URL: stringPtr(plan.OllamaConfig.URL),
			Key: stringPtr(plan.OllamaConfig.Key),
		}
	}

	if plan.AzureOpenAIConfig != nil {
		form.AzureOpenAIConfig = &client.RAGEngineConfigForm{
			URL:     stringPtr(plan.AzureOpenAIConfig.URL),
			Key:     stringPtr(plan.AzureOpenAIConfig.Key),
			Version: stringPtr(plan.AzureOpenAIConfig.Version),
		}
	}

	updated, err := apiClient.SetRAGEmbeddingConfig(ctx, form)
	if err != nil {
		diags.AddError("Update RAG embedding config failed", err.Error())
		return ragEmbeddingConfigModel{}
	}

	return flattenRAGEmbeddingConfig(plan, updated)
}

// flattenRAGEmbeddingConfig records what Open WebUI returned. An engine block
// stays out of state until a configuration names it, because a resource that
// does not manage the block must not report its credentials as managed.
func flattenRAGEmbeddingConfig(prior ragEmbeddingConfigModel, remote *client.RAGEmbeddingConfigForm) ragEmbeddingConfigModel {
	state := ragEmbeddingConfigModel{
		ID:                             types.StringValue("rag_embedding"),
		RAGEmbeddingEngine:             stringValueOrNull(remote.RAGEmbeddingEngine),
		RAGEmbeddingModel:              stringValueOrNull(remote.RAGEmbeddingModel),
		RAGEmbeddingBatchSize:          int64ValueOrNull(remote.RAGEmbeddingBatchSize),
		EnableAsyncEmbedding:           engineBoolValue(remote.EnableAsyncEmbedding),
		RAGEmbeddingConcurrentRequests: int64ValueOrNull(remote.RAGEmbeddingConcurrentRequests),
	}

	if prior.OpenAIConfig != nil && remote.OpenAIConfig != nil {
		state.OpenAIConfig = &ragEngineConfigModel{
			URL: stringValueOrNull(remote.OpenAIConfig.URL),
			Key: stringValueOrNull(remote.OpenAIConfig.Key),
		}
	}

	if prior.OllamaConfig != nil && remote.OllamaConfig != nil {
		state.OllamaConfig = &ragEngineConfigModel{
			URL: stringValueOrNull(remote.OllamaConfig.URL),
			Key: stringValueOrNull(remote.OllamaConfig.Key),
		}
	}

	if prior.AzureOpenAIConfig != nil && remote.AzureOpenAIConfig != nil {
		state.AzureOpenAIConfig = &ragAzureEngineConfigModel{
			URL:     stringValueOrNull(remote.AzureOpenAIConfig.URL),
			Key:     stringValueOrNull(remote.AzureOpenAIConfig.Key),
			Version: stringValueOrNull(remote.AzureOpenAIConfig.Version),
		}
	}

	return state
}

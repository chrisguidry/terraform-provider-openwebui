package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ provider.Provider = &openWebUIProvider{}

// openWebUIProvider defines the provider implementation.
type openWebUIProvider struct {
	version string
}

// providerModel maps provider schema data to Go type.
type providerModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	Token              types.String `tfsdk:"token"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

// New instantiates a new provider.
func New() provider.Provider {
	return &openWebUIProvider{
		version: Version,
	}
}

// Metadata satisfies the provider.Provider interface.
func (p *openWebUIProvider) Metadata(_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openwebui"
	resp.Version = p.version
}

// Schema defines the provider-level schema.
func (p *openWebUIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The **openwebui** provider manages an [Open WebUI](https://openwebui.com) deployment through its REST API. " +
			"It covers Open WebUI v0.11.\n\n" +
			"## Configuration\n\n" +
			"The provider needs the base URL of the instance and an API token from an administrator account. " +
			"Set them in the provider block, or leave them out and set `OPENWEBUI_ENDPOINT` and `OPENWEBUI_TOKEN` in the environment. " +
			"The token is the API key on the Account page of the Open WebUI settings. An administrator token is required, " +
			"because the configuration resources read and write the admin API.\n\n" +
			"## What the provider manages\n\n" +
			"The navigation lists every resource and data source. They fall into four groups.\n\n" +
			"**Content**: `openwebui_knowledge` and `openwebui_knowledge_file`, `openwebui_file`, `openwebui_model`, " +
			"`openwebui_prompt`, `openwebui_skill`, `openwebui_tool`, `openwebui_function`, `openwebui_pipeline`, and the " +
			"`_valves` resources that carry the settings of a tool, a function, or a pipeline.\n\n" +
			"**People and access**: `openwebui_user`, `openwebui_group`, `openwebui_channel`, and " +
			"`openwebui_default_user_permissions` for what a new account may do.\n\n" +
			"**Instance configuration**: one resource for each page of the admin settings, including `openwebui_admin_config`, " +
			"`openwebui_chat_config`, `openwebui_audio_config`, `openwebui_images_config`, `openwebui_rag_config`, " +
			"`openwebui_rag_embedding_config`, `openwebui_code_execution_config`, `openwebui_task_config`, " +
			"`openwebui_subagents_config`, `openwebui_evaluation_config`, `openwebui_ldap_config`, `openwebui_banners_config`, " +
			"`openwebui_suggestions_config`, and `openwebui_models_config`. Each one is a singleton: the instance holds a " +
			"single copy of those settings, so declare the resource once. `openwebui_config_export` reads the whole " +
			"configuration back, and `openwebui_config_import` restores one.\n\n" +
			"**Backends**: `openwebui_ollama_connections` and `openwebui_openai_connections` for model endpoints, " +
			"`openwebui_tool_server` and `openwebui_terminal_server` for external servers, and `openwebui_oauth_client` for " +
			"the credentials they authenticate with. `openwebui_tool_servers_config` writes the whole tool server list at " +
			"once, and `openwebui_connections_config` carries the direct-connection settings. The `_verify` data sources ask " +
			"Open WebUI to connect to an endpoint, and the read fails when it cannot.\n",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Open WebUI instance, e.g. `https://openwebui.example.com`. Can also be set via the `OPENWEBUI_ENDPOINT` environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token used to authenticate requests to the Open WebUI API. Can also be set via the `OPENWEBUI_TOKEN` environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Disable TLS certificate verification. **Not recommended for production use.** Can also be set via the `OPENWEBUI_INSECURE` environment variable, where any non-empty value disables verification.",
			},
		},
	}
}

// Configure prepares the Open WebUI API client for data sources and resources.
func (p *openWebUIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := ""
	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		endpoint = data.Endpoint.ValueString()
	} else if envEndpoint := os.Getenv("OPENWEBUI_ENDPOINT"); envEndpoint != "" {
		endpoint = envEndpoint
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Open WebUI API endpoint",
			"An endpoint must be supplied via the provider configuration or the OPENWEBUI_ENDPOINT environment variable.",
		)
	}

	token := ""
	if !data.Token.IsNull() && !data.Token.IsUnknown() {
		token = data.Token.ValueString()
	} else if envToken := os.Getenv("OPENWEBUI_TOKEN"); envToken != "" {
		token = envToken
	}

	insecure := false
	if !data.InsecureSkipVerify.IsNull() && !data.InsecureSkipVerify.IsUnknown() {
		insecure = data.InsecureSkipVerify.ValueBool()
	} else if os.Getenv("OPENWEBUI_INSECURE") != "" {
		insecure = true
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Open WebUI API token",
			"A valid API token must be supplied via the provider configuration or the OPENWEBUI_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiClient, err := client.NewClient(endpoint, token, insecure)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Open WebUI API client",
			err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Configured Open WebUI provider", map[string]any{
		"endpoint": endpoint,
	})

	resp.ResourceData = apiClient
	resp.DataSourceData = apiClient
}

// Resources defines provider-supported resources.

// Resource and data-source constructors registered from their own files.
// Each resource file appends its constructors in an init function, so
// adding a resource never edits this file.
var (
	registeredResources   []func() resource.Resource
	registeredDataSources []func() datasource.DataSource
)

func (p *openWebUIProvider) Resources(_ context.Context) []func() resource.Resource {
	return append([]func() resource.Resource{
		NewKnowledgeResource,
		NewModelResource,
		NewPromptResource,
		NewGroupResource,
		NewToolResource,
		NewToolValvesResource,
		NewSkillResource,
		NewPipelineResource,
		NewPipelineValvesResource,
		NewFileResource,
		NewKnowledgeFileResource,
		NewConfigImportResource,
		NewConnectionsConfigResource,
		NewToolServersConfigResource,
		NewCodeExecutionConfigResource,
		NewModelsConfigResource,
		NewSuggestionsConfigResource,
		NewBannersConfigResource,
		NewOAuthClientResource,
		NewFunctionResource,
		NewFunctionValvesResource,
		NewOpenAIConnectionsResource,
		NewOllamaConnectionsResource,
	}, registeredResources...)
}

// DataSources defines provider-supported data sources.
func (p *openWebUIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return append([]func() datasource.DataSource{
		NewModelDataSource,
		NewKnowledgeDataSource,
		NewGroupDataSource,
		NewPromptDataSource,
		NewToolDataSource,
		NewSkillDataSource,
		NewPipelineDataSource,
		NewFileDataSource,
		NewFilesDataSource,
		NewConfigExportDataSource,
		NewUserDataSource,
		NewToolServerVerifyDataSource,
		NewOpenAIConnectionVerifyDataSource,
		NewOllamaConnectionVerifyDataSource,
	}, registeredDataSources...)
}

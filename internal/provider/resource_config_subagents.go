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

var _ resource.Resource = &subagentsConfigResource{}
var _ resource.ResourceWithConfigure = &subagentsConfigResource{}
var _ resource.ResourceWithImportState = &subagentsConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewSubagentsConfigResource)
}

// subagentsConfigResource manages the subagent settings.
type subagentsConfigResource struct {
	client *client.Client
}

type subagentsConfigModel struct {
	ID                types.String `tfsdk:"id"`
	Enabled           types.Bool   `tfsdk:"enable_subagents"`
	BackgroundEnabled types.Bool   `tfsdk:"subagents_background_enabled"`
	MaxConcurrent     types.Int64  `tfsdk:"subagents_max_concurrent"`
	MaxAsync          types.Int64  `tfsdk:"subagents_max_async"`
	MaxIterations     types.Int64  `tfsdk:"subagents_max_iterations"`
	MaxOutput         types.Int64  `tfsdk:"subagents_max_output"`
	SystemPrompt      types.String `tfsdk:"subagents_system_prompt"`
}

// NewSubagentsConfigResource constructs a new subagents config resource.
func NewSubagentsConfigResource() resource.Resource {
	return &subagentsConfigResource{}
}

// Metadata sets the resource type name.
func (r *subagentsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subagents_config"
}

// Schema defines the subagents config schema.
func (r *subagentsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the subagent settings for Open WebUI. Subagents let a chat delegate work to a nested agent run. " +
			"Every attribute is required, because Open WebUI writes all seven settings on every update.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_subagents": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether subagents are available at all.",
				MarkdownDescription: "Whether subagents are available at all.",
			},
			"subagents_background_enabled": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether a subagent may keep running in the background after the chat turn ends.",
				MarkdownDescription: "Whether a subagent may keep running in the background after the chat turn ends.",
			},
			"subagents_max_concurrent": schema.Int64Attribute{
				Required:            true,
				Description:         "Maximum number of subagents that run at the same time.",
				MarkdownDescription: "Maximum number of subagents that run at the same time.",
			},
			"subagents_max_async": schema.Int64Attribute{
				Required:            true,
				Description:         "Maximum number of background subagents.",
				MarkdownDescription: "Maximum number of background subagents.",
			},
			"subagents_max_iterations": schema.Int64Attribute{
				Required:            true,
				Description:         "Maximum number of turns one subagent takes before it stops.",
				MarkdownDescription: "Maximum number of turns one subagent takes before it stops.",
			},
			"subagents_max_output": schema.Int64Attribute{
				Required:            true,
				Description:         "Maximum size in characters of the output a subagent returns to its caller.",
				MarkdownDescription: "Maximum size in characters of the output a subagent returns to its caller.",
			},
			"subagents_system_prompt": schema.StringAttribute{
				Required:            true,
				Description:         "System prompt given to every subagent. An empty string leaves the built-in prompt in place.",
				MarkdownDescription: "System prompt given to every subagent. An empty string leaves the built-in prompt in place.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *subagentsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the subagents config.
func (r *subagentsConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	var plan subagentsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySubagentsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the subagents config.
func (r *subagentsConfigResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	config, err := r.client.GetSubagentsConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read subagents config failed", err.Error())
		return
	}

	state := flattenSubagentsConfig(config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the subagents config.
func (r *subagentsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing subagents config.")
		return
	}

	var plan subagentsConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applySubagentsConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves the settings in place.
func (r *subagentsConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *subagentsConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applySubagentsConfig(ctx context.Context, apiClient *client.Client, plan subagentsConfigModel) (subagentsConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.SubagentsConfigForm{
		EnableSubagents:            plan.Enabled.ValueBool(),
		SubagentsBackgroundEnabled: plan.BackgroundEnabled.ValueBool(),
		SubagentsMaxConcurrent:     plan.MaxConcurrent.ValueInt64(),
		SubagentsMaxAsync:          plan.MaxAsync.ValueInt64(),
		SubagentsMaxIterations:     plan.MaxIterations.ValueInt64(),
		SubagentsMaxOutput:         plan.MaxOutput.ValueInt64(),
		SubagentsSystemPrompt:      plan.SystemPrompt.ValueString(),
	}

	updated, err := apiClient.SetSubagentsConfig(ctx, form)
	if err != nil {
		diags.AddError("Update subagents config failed", err.Error())
		return subagentsConfigModel{}, diags
	}

	return flattenSubagentsConfig(updated), diags
}

func flattenSubagentsConfig(config *client.SubagentsConfigForm) subagentsConfigModel {
	return subagentsConfigModel{
		ID:                types.StringValue("subagents"),
		Enabled:           types.BoolValue(config.EnableSubagents),
		BackgroundEnabled: types.BoolValue(config.SubagentsBackgroundEnabled),
		MaxConcurrent:     types.Int64Value(config.SubagentsMaxConcurrent),
		MaxAsync:          types.Int64Value(config.SubagentsMaxAsync),
		MaxIterations:     types.Int64Value(config.SubagentsMaxIterations),
		MaxOutput:         types.Int64Value(config.SubagentsMaxOutput),
		SystemPrompt:      types.StringValue(config.SubagentsSystemPrompt),
	}
}

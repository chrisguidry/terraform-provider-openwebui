package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &chatConfigResource{}
var _ resource.ResourceWithConfigure = &chatConfigResource{}
var _ resource.ResourceWithImportState = &chatConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewChatConfigResource)
}

// chatConfigResource manages context compaction settings.
type chatConfigResource struct {
	client *client.Client
}

type chatConfigModel struct {
	ID                                   types.String `tfsdk:"id"`
	ContextCompactionModel               types.String `tfsdk:"context_compaction_model"`
	EnableContextCompaction              types.Bool   `tfsdk:"enable_context_compaction"`
	ContextCompactionTokenThreshold      types.Int64  `tfsdk:"context_compaction_token_threshold"`
	ContextCompactionTokenCap            types.Int64  `tfsdk:"context_compaction_token_cap"`
	ContextCompactionRetentionPercentage types.Int64  `tfsdk:"context_compaction_retention_percentage"`
	ContextCompactionPromptTemplate      types.String `tfsdk:"context_compaction_prompt_template"`
}

// NewChatConfigResource constructs a new chat config resource.
func NewChatConfigResource() resource.Resource {
	return &chatConfigResource{}
}

// Metadata sets the resource type name.
func (r *chatConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_config"
}

// Schema defines the chat config schema.
func (r *chatConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages context compaction for chats in Open WebUI. Compaction summarises an older part of a conversation once it grows past a token threshold.\n\n" +
			"Do not write these settings through `openwebui_config_import` as well. That path writes the stored keys directly and skips the bounds Open WebUI applies here, so the two would overwrite each other.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"context_compaction_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Model that writes the summary. An empty value uses the chat's own model.",
				MarkdownDescription: "Model that writes the summary. An empty value uses the chat's own model.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_context_compaction": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI compacts a conversation once it passes the token threshold.",
				MarkdownDescription: "Whether Open WebUI compacts a conversation once it passes the token threshold.",
			},
			"context_compaction_token_threshold": schema.Int64Attribute{
				Required:            true,
				Description:         "Token count at which a conversation is compacted. Open WebUI raises a lower value to 1.",
				MarkdownDescription: "Token count at which a conversation is compacted. Open WebUI raises a lower value to 1.",
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"context_compaction_token_cap": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Upper bound on the tokens a compacted conversation keeps. Open WebUI stores the threshold when this is unset, so it never reads back as null.",
				MarkdownDescription: "Upper bound on the tokens a compacted conversation keeps. Open WebUI stores the threshold when this is unset, so it never reads back as null.",
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"context_compaction_retention_percentage": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Percentage of the conversation kept verbatim after compaction. Open WebUI clamps this to between 10 and 50.",
				MarkdownDescription: "Percentage of the conversation kept verbatim after compaction. Open WebUI clamps this to between 10 and 50.",
				Validators:          []validator.Int64{int64validator.Between(10, 50)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"context_compaction_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template that produces the summary.",
				MarkdownDescription: "Prompt template that produces the summary.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *chatConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create writes the chat config.
func (r *chatConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	var plan chatConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyChatConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the chat config.
func (r *chatConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	config, err := r.client.GetChatConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read chat config failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, chatConfigToModel(config))...)
}

// Update writes the chat config.
func (r *chatConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}

	var plan chatConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyChatConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *chatConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing chat config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *chatConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyChatConfig writes the plan. The route takes all six values at once, and
// three of them are not nullable on the wire, so an attribute the plan leaves
// unknown carries the value the server holds today.
func applyChatConfig(ctx context.Context, apiClient *client.Client, plan chatConfigModel) (chatConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.ChatConfigForm{
		ContextCompactionModel:          stringPtr(plan.ContextCompactionModel),
		EnableContextCompaction:         plan.EnableContextCompaction.ValueBool(),
		ContextCompactionTokenThreshold: plan.ContextCompactionTokenThreshold.ValueInt64(),
		ContextCompactionTokenCap:       int64Ptr(plan.ContextCompactionTokenCap),
		ContextCompactionPromptTemplate: plan.ContextCompactionPromptTemplate.ValueString(),
	}

	if plan.ContextCompactionRetentionPercentage.IsNull() || plan.ContextCompactionRetentionPercentage.IsUnknown() {
		current, err := apiClient.GetChatConfig(ctx)
		if err != nil {
			diags.AddError("Read chat config failed", err.Error())
			return chatConfigModel{}, diags
		}
		form.ContextCompactionRetentionPercentage = current.ContextCompactionRetentionPercentage
	} else {
		form.ContextCompactionRetentionPercentage = plan.ContextCompactionRetentionPercentage.ValueInt64()
	}

	updated, err := apiClient.SetChatConfig(ctx, form)
	if err != nil {
		diags.AddError("Update chat config failed", err.Error())
		return chatConfigModel{}, diags
	}

	return chatConfigToModel(updated), diags
}

func chatConfigToModel(config *client.ChatConfigForm) chatConfigModel {
	model := types.StringValue("")
	if config.ContextCompactionModel != nil {
		model = types.StringValue(*config.ContextCompactionModel)
	}

	tokenCap := types.Int64Null()
	if config.ContextCompactionTokenCap != nil {
		tokenCap = types.Int64Value(*config.ContextCompactionTokenCap)
	}

	return chatConfigModel{
		ID:                                   types.StringValue("chat"),
		ContextCompactionModel:               model,
		EnableContextCompaction:              types.BoolValue(config.EnableContextCompaction),
		ContextCompactionTokenThreshold:      types.Int64Value(config.ContextCompactionTokenThreshold),
		ContextCompactionTokenCap:            tokenCap,
		ContextCompactionRetentionPercentage: types.Int64Value(config.ContextCompactionRetentionPercentage),
		ContextCompactionPromptTemplate:      types.StringValue(config.ContextCompactionPromptTemplate),
	}
}

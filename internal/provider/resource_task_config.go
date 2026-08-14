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

var _ resource.Resource = &taskConfigResource{}
var _ resource.ResourceWithConfigure = &taskConfigResource{}
var _ resource.ResourceWithImportState = &taskConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewTaskConfigResource)
}

// taskConfigResource manages the background task settings: the task model and
// the prompt templates Open WebUI uses to title chats, tag them, autocomplete,
// build search queries, suggest follow ups, call tools, and drive voice mode.
type taskConfigResource struct {
	client *client.Client
}

type taskConfigModel struct {
	ID                                   types.String `tfsdk:"id"`
	TaskModel                            types.String `tfsdk:"task_model"`
	TaskModelExternal                    types.String `tfsdk:"task_model_external"`
	EnableTitleGeneration                types.Bool   `tfsdk:"enable_title_generation"`
	TitleGenerationPromptTemplate        types.String `tfsdk:"title_generation_prompt_template"`
	ImagePromptGenerationPromptTemplate  types.String `tfsdk:"image_prompt_generation_prompt_template"`
	EnableAutocompleteGeneration         types.Bool   `tfsdk:"enable_autocomplete_generation"`
	AutocompleteGenerationInputMaxLength types.Int64  `tfsdk:"autocomplete_generation_input_max_length"`
	AutocompleteGenerationPromptTemplate types.String `tfsdk:"autocomplete_generation_prompt_template"`
	TagsGenerationPromptTemplate         types.String `tfsdk:"tags_generation_prompt_template"`
	FollowUpGenerationPromptTemplate     types.String `tfsdk:"follow_up_generation_prompt_template"`
	EnableFollowUpGeneration             types.Bool   `tfsdk:"enable_follow_up_generation"`
	EnableTagsGeneration                 types.Bool   `tfsdk:"enable_tags_generation"`
	EnableSearchQueryGeneration          types.Bool   `tfsdk:"enable_search_query_generation"`
	EnableRetrievalQueryGeneration       types.Bool   `tfsdk:"enable_retrieval_query_generation"`
	QueryGenerationPromptTemplate        types.String `tfsdk:"query_generation_prompt_template"`
	ToolsFunctionCallingPromptTemplate   types.String `tfsdk:"tools_function_calling_prompt_template"`
	EnableVoiceModePrompt                types.Bool   `tfsdk:"enable_voice_mode_prompt"`
	VoiceModePromptTemplate              types.String `tfsdk:"voice_mode_prompt_template"`
}

// NewTaskConfigResource constructs a new task config resource.
func NewTaskConfigResource() resource.Resource {
	return &taskConfigResource{}
}

// Metadata sets the resource type name.
func (r *taskConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_task_config"
}

// Schema defines the task config schema.
func (r *taskConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	const templateNote = " Set it to an empty string to use the template built into Open WebUI."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the task settings for Open WebUI: the model that runs background tasks and the prompt templates for " +
			"chat titles, tags, autocomplete, search queries, follow ups, tool calling, and voice mode. " +
			"Open WebUI writes all eighteen settings on every update, so every attribute of this resource is required. " +
			"An empty template string is a real value that means \"use the template built into Open WebUI\".",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"task_model": schema.StringAttribute{
				Optional:            true,
				Description:         "Model id that runs tasks for local models. Null makes Open WebUI use the model of the chat itself.",
				MarkdownDescription: "Model id that runs tasks for local models. Null makes Open WebUI use the model of the chat itself.",
			},
			"task_model_external": schema.StringAttribute{
				Optional:            true,
				Description:         "Model id that runs tasks for external models. Null makes Open WebUI use the model of the chat itself.",
				MarkdownDescription: "Model id that runs tasks for external models. Null makes Open WebUI use the model of the chat itself.",
			},
			"enable_title_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI generates a title for each chat.",
				MarkdownDescription: "Whether Open WebUI generates a title for each chat.",
			},
			"title_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template for chat titles." + templateNote,
				MarkdownDescription: "Prompt template for chat titles." + templateNote,
			},
			"image_prompt_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template that turns a chat into an image generation prompt." + templateNote,
				MarkdownDescription: "Prompt template that turns a chat into an image generation prompt." + templateNote,
			},
			"enable_autocomplete_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI completes the message a user is typing.",
				MarkdownDescription: "Whether Open WebUI completes the message a user is typing.",
			},
			"autocomplete_generation_input_max_length": schema.Int64Attribute{
				Required:            true,
				Description:         "Maximum length in characters of the input sent for autocompletion. A negative value removes the limit.",
				MarkdownDescription: "Maximum length in characters of the input sent for autocompletion. A negative value removes the limit.",
			},
			"autocomplete_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template for autocompletion." + templateNote,
				MarkdownDescription: "Prompt template for autocompletion." + templateNote,
			},
			"tags_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template for chat tags." + templateNote,
				MarkdownDescription: "Prompt template for chat tags." + templateNote,
			},
			"follow_up_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template for follow-up suggestions." + templateNote,
				MarkdownDescription: "Prompt template for follow-up suggestions." + templateNote,
			},
			"enable_follow_up_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI suggests follow-up questions.",
				MarkdownDescription: "Whether Open WebUI suggests follow-up questions.",
			},
			"enable_tags_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI generates tags for each chat.",
				MarkdownDescription: "Whether Open WebUI generates tags for each chat.",
			},
			"enable_search_query_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI writes a web search query from the chat.",
				MarkdownDescription: "Whether Open WebUI writes a web search query from the chat.",
			},
			"enable_retrieval_query_generation": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI writes a knowledge retrieval query from the chat.",
				MarkdownDescription: "Whether Open WebUI writes a knowledge retrieval query from the chat.",
			},
			"query_generation_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template for search and retrieval queries." + templateNote,
				MarkdownDescription: "Prompt template for search and retrieval queries." + templateNote,
			},
			"tools_function_calling_prompt_template": schema.StringAttribute{
				Required:            true,
				Description:         "Prompt template that asks a model without native tool calling to pick a tool." + templateNote,
				MarkdownDescription: "Prompt template that asks a model without native tool calling to pick a tool." + templateNote,
			},
			"enable_voice_mode_prompt": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether Open WebUI adds a voice mode system prompt to spoken conversations.",
				MarkdownDescription: "Whether Open WebUI adds a voice mode system prompt to spoken conversations.",
			},
			"voice_mode_prompt_template": schema.StringAttribute{
				Optional:            true,
				Description:         "Prompt template for voice mode." + templateNote,
				MarkdownDescription: "Prompt template for voice mode." + templateNote,
			},
		},
	}
}

// Configure assigns the API client.
func (r *taskConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the task config.
func (r *taskConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing task config.")
		return
	}

	var plan taskConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyTaskConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the task config.
func (r *taskConfigResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing task config.")
		return
	}

	config, err := r.client.GetTaskConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read task config failed", err.Error())
		return
	}

	state := flattenTaskConfig(config)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the task config.
func (r *taskConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing task config.")
		return
	}

	var plan taskConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyTaskConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves the settings in place.
func (r *taskConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *taskConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyTaskConfig(ctx context.Context, apiClient *client.Client, plan taskConfigModel) (taskConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	form := client.TaskConfigForm{
		TaskModel:                            stringPtr(plan.TaskModel),
		TaskModelExternal:                    stringPtr(plan.TaskModelExternal),
		EnableTitleGeneration:                plan.EnableTitleGeneration.ValueBool(),
		TitleGenerationPromptTemplate:        plan.TitleGenerationPromptTemplate.ValueString(),
		ImagePromptGenerationPromptTemplate:  plan.ImagePromptGenerationPromptTemplate.ValueString(),
		EnableAutocompleteGeneration:         plan.EnableAutocompleteGeneration.ValueBool(),
		AutocompleteGenerationInputMaxLength: plan.AutocompleteGenerationInputMaxLength.ValueInt64(),
		AutocompleteGenerationPromptTemplate: plan.AutocompleteGenerationPromptTemplate.ValueString(),
		TagsGenerationPromptTemplate:         plan.TagsGenerationPromptTemplate.ValueString(),
		FollowUpGenerationPromptTemplate:     plan.FollowUpGenerationPromptTemplate.ValueString(),
		EnableFollowUpGeneration:             plan.EnableFollowUpGeneration.ValueBool(),
		EnableTagsGeneration:                 plan.EnableTagsGeneration.ValueBool(),
		EnableSearchQueryGeneration:          plan.EnableSearchQueryGeneration.ValueBool(),
		EnableRetrievalQueryGeneration:       plan.EnableRetrievalQueryGeneration.ValueBool(),
		QueryGenerationPromptTemplate:        plan.QueryGenerationPromptTemplate.ValueString(),
		ToolsFunctionCallingPromptTemplate:   plan.ToolsFunctionCallingPromptTemplate.ValueString(),
		EnableVoiceModePrompt:                plan.EnableVoiceModePrompt.ValueBool(),
		VoiceModePromptTemplate:              stringPtr(plan.VoiceModePromptTemplate),
	}

	updated, err := apiClient.SetTaskConfig(ctx, form)
	if err != nil {
		diags.AddError("Update task config failed", err.Error())
		return taskConfigModel{}, diags
	}

	return flattenTaskConfig(updated), diags
}

func flattenTaskConfig(config *client.TaskConfigForm) taskConfigModel {
	return taskConfigModel{
		ID:                                   types.StringValue("task_config"),
		TaskModel:                            stringValueOrNull(config.TaskModel),
		TaskModelExternal:                    stringValueOrNull(config.TaskModelExternal),
		EnableTitleGeneration:                types.BoolValue(config.EnableTitleGeneration),
		TitleGenerationPromptTemplate:        types.StringValue(config.TitleGenerationPromptTemplate),
		ImagePromptGenerationPromptTemplate:  types.StringValue(config.ImagePromptGenerationPromptTemplate),
		EnableAutocompleteGeneration:         types.BoolValue(config.EnableAutocompleteGeneration),
		AutocompleteGenerationInputMaxLength: types.Int64Value(config.AutocompleteGenerationInputMaxLength),
		AutocompleteGenerationPromptTemplate: types.StringValue(config.AutocompleteGenerationPromptTemplate),
		TagsGenerationPromptTemplate:         types.StringValue(config.TagsGenerationPromptTemplate),
		FollowUpGenerationPromptTemplate:     types.StringValue(config.FollowUpGenerationPromptTemplate),
		EnableFollowUpGeneration:             types.BoolValue(config.EnableFollowUpGeneration),
		EnableTagsGeneration:                 types.BoolValue(config.EnableTagsGeneration),
		EnableSearchQueryGeneration:          types.BoolValue(config.EnableSearchQueryGeneration),
		EnableRetrievalQueryGeneration:       types.BoolValue(config.EnableRetrievalQueryGeneration),
		QueryGenerationPromptTemplate:        types.StringValue(config.QueryGenerationPromptTemplate),
		ToolsFunctionCallingPromptTemplate:   types.StringValue(config.ToolsFunctionCallingPromptTemplate),
		EnableVoiceModePrompt:                types.BoolValue(config.EnableVoiceModePrompt),
		VoiceModePromptTemplate:              stringValueOrNull(config.VoiceModePromptTemplate),
	}
}

package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

func init() {
	registeredResources = append(registeredResources, NewImagesConfigResource)
}

var _ resource.Resource = &imagesConfigResource{}
var _ resource.ResourceWithConfigure = &imagesConfigResource{}
var _ resource.ResourceWithImportState = &imagesConfigResource{}

// imageSizePattern mirrors the check the update route runs before it stores a
// size (backend/open_webui/routers/images.py, lines 288 to 293).
var imageSizePattern = regexp.MustCompile(`^(auto|\d+x\d+)?$`)

// imagesConfigResource manages image generation and image editing settings.
type imagesConfigResource struct {
	client *client.Client
}

// imagesConfigModel mirrors ImagesConfig in backend/open_webui/routers/images.py.
type imagesConfigModel struct {
	ID                             types.String `tfsdk:"id"`
	EnableImageGeneration          types.Bool   `tfsdk:"enable_image_generation"`
	EnableImagePromptGeneration    types.Bool   `tfsdk:"enable_image_prompt_generation"`
	ImageGenerationEngine          types.String `tfsdk:"image_generation_engine"`
	ImageGenerationModel           types.String `tfsdk:"image_generation_model"`
	ImageSize                      types.String `tfsdk:"image_size"`
	ImageSteps                     types.Int64  `tfsdk:"image_steps"`
	ImagesOpenAIAPIBaseURL         types.String `tfsdk:"images_openai_api_base_url"`
	ImagesOpenAIAPIKey             types.String `tfsdk:"images_openai_api_key"`
	ImagesOpenAIAPIVersion         types.String `tfsdk:"images_openai_api_version"`
	ImagesOpenAIAPIParams          types.String `tfsdk:"images_openai_api_params"`
	Automatic1111BaseURL           types.String `tfsdk:"automatic1111_base_url"`
	Automatic1111APIAuth           types.String `tfsdk:"automatic1111_api_auth"`
	Automatic1111Params            types.String `tfsdk:"automatic1111_params"`
	ComfyUIBaseURL                 types.String `tfsdk:"comfyui_base_url"`
	ComfyUIAPIKey                  types.String `tfsdk:"comfyui_api_key"`
	ComfyUIWorkflow                types.String `tfsdk:"comfyui_workflow"`
	ComfyUIWorkflowNodes           types.String `tfsdk:"comfyui_workflow_nodes"`
	ImagesGeminiAPIBaseURL         types.String `tfsdk:"images_gemini_api_base_url"`
	ImagesGeminiAPIKey             types.String `tfsdk:"images_gemini_api_key"`
	ImagesGeminiEndpointMethod     types.String `tfsdk:"images_gemini_endpoint_method"`
	EnableImageEdit                types.Bool   `tfsdk:"enable_image_edit"`
	ImageEditEngine                types.String `tfsdk:"image_edit_engine"`
	ImageEditModel                 types.String `tfsdk:"image_edit_model"`
	ImageEditSize                  types.String `tfsdk:"image_edit_size"`
	ImagesEditOpenAIAPIBaseURL     types.String `tfsdk:"images_edit_openai_api_base_url"`
	ImagesEditOpenAIAPIKey         types.String `tfsdk:"images_edit_openai_api_key"`
	ImagesEditOpenAIAPIVersion     types.String `tfsdk:"images_edit_openai_api_version"`
	ImagesEditGeminiAPIBaseURL     types.String `tfsdk:"images_edit_gemini_api_base_url"`
	ImagesEditGeminiAPIKey         types.String `tfsdk:"images_edit_gemini_api_key"`
	ImagesEditComfyUIBaseURL       types.String `tfsdk:"images_edit_comfyui_base_url"`
	ImagesEditComfyUIAPIKey        types.String `tfsdk:"images_edit_comfyui_api_key"`
	ImagesEditComfyUIWorkflow      types.String `tfsdk:"images_edit_comfyui_workflow"`
	ImagesEditComfyUIWorkflowNodes types.String `tfsdk:"images_edit_comfyui_workflow_nodes"`
}

// NewImagesConfigResource constructs a new images config resource.
func NewImagesConfigResource() resource.Resource {
	return &imagesConfigResource{}
}

// Metadata sets the resource type name.
func (r *imagesConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images_config"
}

// Schema defines the images config schema.
func (r *imagesConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's image generation and image editing engines: OpenAI, Automatic1111, ComfyUI, and Gemini.\n\n" +
			"`POST /api/v1/images/config/update` writes every key it receives, so this resource reads the current configuration and " +
			"overlays the attributes the plan names before it writes. A successful apply does not prove the engine is reachable: the " +
			"update route asks Automatic1111 to load the model and swallows the error when the host does not answer.\n\n" +
			"`user.permissions` is in the router's key map but not in its form, so this router never touches user permissions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_image_generation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_IMAGE_GENERATION`. Stored as `image_generation.enable`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_IMAGE_GENERATION`. Stored as `image_generation.enable`.",
			},
			"enable_image_prompt_generation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_IMAGE_PROMPT_GENERATION`. Stored as `image_generation.prompt.enable`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_IMAGE_PROMPT_GENERATION`. Stored as `image_generation.prompt.enable`.",
			},
			"image_generation_engine": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGE_GENERATION_ENGINE`. Stored as `image_generation.engine`.",
				MarkdownDescription: "Open WebUI setting `IMAGE_GENERATION_ENGINE`. Stored as `image_generation.engine`.",
			},
			"image_generation_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGE_GENERATION_MODEL`. Stored as `image_generation.model`.",
				MarkdownDescription: "Open WebUI setting `IMAGE_GENERATION_MODEL`. Stored as `image_generation.model`.",
			},
			"image_size": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Generated image size, `WIDTHxHEIGHT`, or `auto`, or the empty string. Open WebUI accepts `auto` only for models its `IMAGE_AUTO_SIZE_MODELS_REGEX_PATTERN` matches. Stored as `image_generation.size`.",
				MarkdownDescription: "Generated image size, `WIDTHxHEIGHT`, or `auto`, or the empty string. Open WebUI accepts `auto` only for models its `IMAGE_AUTO_SIZE_MODELS_REGEX_PATTERN` matches. Stored as `image_generation.size`.",
				Validators:          []validator.String{stringvalidator.RegexMatches(imageSizePattern, "must be WIDTHxHEIGHT, auto, or empty")},
			},
			"image_steps": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Diffusion steps per image. Must not be negative. Stored as `image_generation.steps`.",
				MarkdownDescription: "Diffusion steps per image. Must not be negative. Stored as `image_generation.steps`.",
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"images_openai_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_OPENAI_API_BASE_URL`. Stored as `image_generation.openai.api_base_url`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_OPENAI_API_BASE_URL`. Stored as `image_generation.openai.api_base_url`.",
			},
			"images_openai_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `IMAGES_OPENAI_API_KEY`. Stored as `image_generation.openai.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `IMAGES_OPENAI_API_KEY`. Stored as `image_generation.openai.api_key`. Sensitive.",
			},
			"images_openai_api_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_OPENAI_API_VERSION`. Stored as `image_generation.openai.api_version`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_OPENAI_API_VERSION`. Stored as `image_generation.openai.api_version`.",
			},
			"images_openai_api_params": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of extra parameters passed to the OpenAI image API. Stored as `image_generation.openai.params`.",
				MarkdownDescription: "JSON object of extra parameters passed to the OpenAI image API. Stored as `image_generation.openai.params`.",
			},
			"automatic1111_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `AUTOMATIC1111_BASE_URL`. Stored as `image_generation.automatic1111.base_url`.",
				MarkdownDescription: "Open WebUI setting `AUTOMATIC1111_BASE_URL`. Stored as `image_generation.automatic1111.base_url`.",
			},
			"automatic1111_api_auth": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Automatic1111 credentials as `user:password`. Open WebUI base64-encodes them at call time. Stored as `image_generation.automatic1111.api_auth`. Sensitive.",
				MarkdownDescription: "Automatic1111 credentials as `user:password`. Open WebUI base64-encodes them at call time. Stored as `image_generation.automatic1111.api_auth`. Sensitive.",
			},
			"automatic1111_params": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON object of extra parameters passed to Automatic1111. Stored as `image_generation.automatic1111.api_params`.",
				MarkdownDescription: "JSON object of extra parameters passed to Automatic1111. Stored as `image_generation.automatic1111.api_params`.",
			},
			"comfyui_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Base URL of the ComfyUI server. Open WebUI strips leading and trailing slashes before it stores the value, and Terraform keeps the form the configuration wrote. Stored as `image_generation.comfyui.base_url`.",
				MarkdownDescription: "Base URL of the ComfyUI server. Open WebUI strips leading and trailing slashes before it stores the value, and Terraform keeps the form the configuration wrote. Stored as `image_generation.comfyui.base_url`.",
			},
			"comfyui_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `COMFYUI_API_KEY`. Stored as `image_generation.comfyui.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `COMFYUI_API_KEY`. Stored as `image_generation.comfyui.api_key`. Sensitive.",
			},
			"comfyui_workflow": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `COMFYUI_WORKFLOW`. Stored as `image_generation.comfyui.workflow`.",
				MarkdownDescription: "Open WebUI setting `COMFYUI_WORKFLOW`. Stored as `image_generation.comfyui.workflow`.",
			},
			"comfyui_workflow_nodes": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON array of ComfyUI workflow node bindings. Stored as `image_generation.comfyui.nodes`.",
				MarkdownDescription: "JSON array of ComfyUI workflow node bindings. Stored as `image_generation.comfyui.nodes`.",
			},
			"images_gemini_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_GEMINI_API_BASE_URL`. Stored as `image_generation.gemini.api_base_url`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_GEMINI_API_BASE_URL`. Stored as `image_generation.gemini.api_base_url`.",
			},
			"images_gemini_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `IMAGES_GEMINI_API_KEY`. Stored as `image_generation.gemini.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `IMAGES_GEMINI_API_KEY`. Stored as `image_generation.gemini.api_key`. Sensitive.",
			},
			"images_gemini_endpoint_method": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_GEMINI_ENDPOINT_METHOD`. Stored as `image_generation.gemini.endpoint_method`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_GEMINI_ENDPOINT_METHOD`. Stored as `image_generation.gemini.endpoint_method`.",
			},
			"enable_image_edit": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `ENABLE_IMAGE_EDIT`. Stored as `images.edit.enable`.",
				MarkdownDescription: "Open WebUI setting `ENABLE_IMAGE_EDIT`. Stored as `images.edit.enable`.",
			},
			"image_edit_engine": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGE_EDIT_ENGINE`. Stored as `images.edit.engine`.",
				MarkdownDescription: "Open WebUI setting `IMAGE_EDIT_ENGINE`. Stored as `images.edit.engine`.",
			},
			"image_edit_model": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGE_EDIT_MODEL`. Stored as `images.edit.model`.",
				MarkdownDescription: "Open WebUI setting `IMAGE_EDIT_MODEL`. Stored as `images.edit.model`.",
			},
			"image_edit_size": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGE_EDIT_SIZE`. Stored as `images.edit.size`.",
				MarkdownDescription: "Open WebUI setting `IMAGE_EDIT_SIZE`. Stored as `images.edit.size`.",
			},
			"images_edit_openai_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_EDIT_OPENAI_API_BASE_URL`. Stored as `images.edit.openai.api_base_url`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_OPENAI_API_BASE_URL`. Stored as `images.edit.openai.api_base_url`.",
			},
			"images_edit_openai_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `IMAGES_EDIT_OPENAI_API_KEY`. Stored as `images.edit.openai.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_OPENAI_API_KEY`. Stored as `images.edit.openai.api_key`. Sensitive.",
			},
			"images_edit_openai_api_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_EDIT_OPENAI_API_VERSION`. Stored as `images.edit.openai.api_version`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_OPENAI_API_VERSION`. Stored as `images.edit.openai.api_version`.",
			},
			"images_edit_gemini_api_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_EDIT_GEMINI_API_BASE_URL`. Stored as `images.edit.gemini.api_base_url`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_GEMINI_API_BASE_URL`. Stored as `images.edit.gemini.api_base_url`.",
			},
			"images_edit_gemini_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `IMAGES_EDIT_GEMINI_API_KEY`. Stored as `images.edit.gemini.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_GEMINI_API_KEY`. Stored as `images.edit.gemini.api_key`. Sensitive.",
			},
			"images_edit_comfyui_base_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Base URL of the ComfyUI server used for image editing. Open WebUI strips leading and trailing slashes before it stores the value, and Terraform keeps the form the configuration wrote. Stored as `images.edit.comfyui.base_url`.",
				MarkdownDescription: "Base URL of the ComfyUI server used for image editing. Open WebUI strips leading and trailing slashes before it stores the value, and Terraform keeps the form the configuration wrote. Stored as `images.edit.comfyui.base_url`.",
			},
			"images_edit_comfyui_api_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Open WebUI setting `IMAGES_EDIT_COMFYUI_API_KEY`. Stored as `images.edit.comfyui.api_key`. Sensitive.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_COMFYUI_API_KEY`. Stored as `images.edit.comfyui.api_key`. Sensitive.",
			},
			"images_edit_comfyui_workflow": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Open WebUI setting `IMAGES_EDIT_COMFYUI_WORKFLOW`. Stored as `images.edit.comfyui.workflow`.",
				MarkdownDescription: "Open WebUI setting `IMAGES_EDIT_COMFYUI_WORKFLOW`. Stored as `images.edit.comfyui.workflow`.",
			},
			"images_edit_comfyui_workflow_nodes": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "JSON array of ComfyUI workflow node bindings used for image editing. Stored as `images.edit.comfyui.nodes`.",
				MarkdownDescription: "JSON array of ComfyUI workflow node bindings used for image editing. Stored as `images.edit.comfyui.nodes`.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *imagesConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the planned the images config.
func (r *imagesConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the images config.")
		return
	}

	var plan imagesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyImagesConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the the images config from Open WebUI.
func (r *imagesConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the images config.")
		return
	}

	var recorded imagesConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := readImagesConfig(ctx, r.client, recorded, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the planned the images config.
func (r *imagesConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the images config.")
		return
	}

	var plan imagesConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyImagesConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves Open WebUI's settings alone.
// There is no route that restores a default, and a config surface outlives the
// Terraform resource that describes it.
func (r *imagesConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *imagesConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyImagesConfig overlays the planned attributes on the current
// configuration and writes the whole form. Every field of ImagesConfig is
// required and the route writes all of them, so a request built from the plan
// alone would replace every unset key with a null.
func applyImagesConfig(ctx context.Context, apiClient *client.Client, plan imagesConfigModel, diags *diag.Diagnostics) imagesConfigModel {
	current, err := apiClient.GetImagesConfig(ctx)
	if err != nil {
		diags.AddError("Read images config failed", err.Error())
		return imagesConfigModel{}
	}

	form := expandImagesConfig(plan, current, diags)
	if diags.HasError() {
		return imagesConfigModel{}
	}

	updated, err := apiClient.SetImagesConfig(ctx, form)
	if err != nil {
		diags.AddError("Update images config failed", err.Error())
		return imagesConfigModel{}
	}

	return flattenImagesConfig(plan, updated, diags)
}

// readImagesConfig refreshes the recorded state from Open WebUI.
func readImagesConfig(ctx context.Context, apiClient *client.Client, recorded imagesConfigModel, diags *diag.Diagnostics) imagesConfigModel {
	current, err := apiClient.GetImagesConfig(ctx)
	if err != nil {
		diags.AddError("Read images config failed", err.Error())
		return imagesConfigModel{}
	}

	return flattenImagesConfig(recorded, current, diags)
}

// expandImagesConfig builds the request form, taking each field from the plan
// when the plan carries a value and from the current configuration otherwise.
func expandImagesConfig(plan imagesConfigModel, current *client.ImagesConfigForm, diags *diag.Diagnostics) client.ImagesConfigForm {
	root := path.Empty()
	form := client.ImagesConfigForm{}
	form.EnableImageGeneration = engineBool(plan.EnableImageGeneration, current.EnableImageGeneration)
	form.EnableImagePromptGeneration = engineBool(plan.EnableImagePromptGeneration, current.EnableImagePromptGeneration)
	form.ImageGenerationEngine = engineString(plan.ImageGenerationEngine, current.ImageGenerationEngine)
	form.ImageGenerationModel = engineString(plan.ImageGenerationModel, current.ImageGenerationModel)
	form.ImageSize = engineString(plan.ImageSize, current.ImageSize)
	form.ImageSteps = engineInt64(plan.ImageSteps, current.ImageSteps)
	form.ImagesOpenAIAPIBaseURL = engineString(plan.ImagesOpenAIAPIBaseURL, current.ImagesOpenAIAPIBaseURL)
	form.ImagesOpenAIAPIKey = engineString(plan.ImagesOpenAIAPIKey, current.ImagesOpenAIAPIKey)
	form.ImagesOpenAIAPIVersion = engineString(plan.ImagesOpenAIAPIVersion, current.ImagesOpenAIAPIVersion)
	form.ImagesOpenAIAPIParams = engineJSON(plan.ImagesOpenAIAPIParams, current.ImagesOpenAIAPIParams, root.AtName("images_openai_api_params"), diags)
	form.Automatic1111BaseURL = engineString(plan.Automatic1111BaseURL, current.Automatic1111BaseURL)
	form.Automatic1111APIAuth = engineString(plan.Automatic1111APIAuth, current.Automatic1111APIAuth)
	form.Automatic1111Params = engineJSON(plan.Automatic1111Params, current.Automatic1111Params, root.AtName("automatic1111_params"), diags)
	form.ComfyUIBaseURL = engineString(plan.ComfyUIBaseURL, current.ComfyUIBaseURL)
	form.ComfyUIAPIKey = engineString(plan.ComfyUIAPIKey, current.ComfyUIAPIKey)
	form.ComfyUIWorkflow = engineString(plan.ComfyUIWorkflow, current.ComfyUIWorkflow)
	form.ComfyUIWorkflowNodes = engineJSON(plan.ComfyUIWorkflowNodes, current.ComfyUIWorkflowNodes, root.AtName("comfyui_workflow_nodes"), diags)
	form.ImagesGeminiAPIBaseURL = engineString(plan.ImagesGeminiAPIBaseURL, current.ImagesGeminiAPIBaseURL)
	form.ImagesGeminiAPIKey = engineString(plan.ImagesGeminiAPIKey, current.ImagesGeminiAPIKey)
	form.ImagesGeminiEndpointMethod = engineString(plan.ImagesGeminiEndpointMethod, current.ImagesGeminiEndpointMethod)
	form.EnableImageEdit = engineBool(plan.EnableImageEdit, current.EnableImageEdit)
	form.ImageEditEngine = engineString(plan.ImageEditEngine, current.ImageEditEngine)
	form.ImageEditModel = engineString(plan.ImageEditModel, current.ImageEditModel)
	form.ImageEditSize = engineString(plan.ImageEditSize, current.ImageEditSize)
	form.ImagesEditOpenAIAPIBaseURL = engineString(plan.ImagesEditOpenAIAPIBaseURL, current.ImagesEditOpenAIAPIBaseURL)
	form.ImagesEditOpenAIAPIKey = engineString(plan.ImagesEditOpenAIAPIKey, current.ImagesEditOpenAIAPIKey)
	form.ImagesEditOpenAIAPIVersion = engineString(plan.ImagesEditOpenAIAPIVersion, current.ImagesEditOpenAIAPIVersion)
	form.ImagesEditGeminiAPIBaseURL = engineString(plan.ImagesEditGeminiAPIBaseURL, current.ImagesEditGeminiAPIBaseURL)
	form.ImagesEditGeminiAPIKey = engineString(plan.ImagesEditGeminiAPIKey, current.ImagesEditGeminiAPIKey)
	form.ImagesEditComfyUIBaseURL = engineString(plan.ImagesEditComfyUIBaseURL, current.ImagesEditComfyUIBaseURL)
	form.ImagesEditComfyUIAPIKey = engineString(plan.ImagesEditComfyUIAPIKey, current.ImagesEditComfyUIAPIKey)
	form.ImagesEditComfyUIWorkflow = engineString(plan.ImagesEditComfyUIWorkflow, current.ImagesEditComfyUIWorkflow)
	form.ImagesEditComfyUIWorkflowNodes = engineJSON(plan.ImagesEditComfyUIWorkflowNodes, current.ImagesEditComfyUIWorkflowNodes, root.AtName("images_edit_comfyui_workflow_nodes"), diags)

	// Open WebUI stores both ComfyUI base URLs without their slashes
	// (backend/open_webui/routers/images.py, lines 302 to 303).
	form.ComfyUIBaseURL = engineTrimmedURL(form.ComfyUIBaseURL)
	form.ImagesEditComfyUIBaseURL = engineTrimmedURL(form.ImagesEditComfyUIBaseURL)

	return form
}

// flattenImagesConfig records the configuration Open WebUI returned.
func flattenImagesConfig(prior imagesConfigModel, remote *client.ImagesConfigForm, diags *diag.Diagnostics) imagesConfigModel {
	state := imagesConfigModel{ID: types.StringValue("images")}
	state.EnableImageGeneration = engineBoolValue(remote.EnableImageGeneration)
	state.EnableImagePromptGeneration = engineBoolValue(remote.EnableImagePromptGeneration)
	state.ImageGenerationEngine = stringValueOrNull(remote.ImageGenerationEngine)
	state.ImageGenerationModel = stringValueOrNull(remote.ImageGenerationModel)
	state.ImageSize = stringValueOrNull(remote.ImageSize)
	state.ImageSteps = int64ValueOrNull(remote.ImageSteps)
	state.ImagesOpenAIAPIBaseURL = stringValueOrNull(remote.ImagesOpenAIAPIBaseURL)
	state.ImagesOpenAIAPIKey = stringValueOrNull(remote.ImagesOpenAIAPIKey)
	state.ImagesOpenAIAPIVersion = stringValueOrNull(remote.ImagesOpenAIAPIVersion)
	state.ImagesOpenAIAPIParams = engineJSONValue(prior.ImagesOpenAIAPIParams, remote.ImagesOpenAIAPIParams, "images_openai_api_params", diags)
	state.Automatic1111BaseURL = stringValueOrNull(remote.Automatic1111BaseURL)
	state.Automatic1111APIAuth = stringValueOrNull(remote.Automatic1111APIAuth)
	state.Automatic1111Params = engineJSONValue(prior.Automatic1111Params, remote.Automatic1111Params, "automatic1111_params", diags)
	state.ComfyUIBaseURL = engineTrimmedURLValue(prior.ComfyUIBaseURL, remote.ComfyUIBaseURL)
	state.ComfyUIAPIKey = stringValueOrNull(remote.ComfyUIAPIKey)
	state.ComfyUIWorkflow = stringValueOrNull(remote.ComfyUIWorkflow)
	state.ComfyUIWorkflowNodes = engineJSONValue(prior.ComfyUIWorkflowNodes, remote.ComfyUIWorkflowNodes, "comfyui_workflow_nodes", diags)
	state.ImagesGeminiAPIBaseURL = stringValueOrNull(remote.ImagesGeminiAPIBaseURL)
	state.ImagesGeminiAPIKey = stringValueOrNull(remote.ImagesGeminiAPIKey)
	state.ImagesGeminiEndpointMethod = stringValueOrNull(remote.ImagesGeminiEndpointMethod)
	state.EnableImageEdit = engineBoolValue(remote.EnableImageEdit)
	state.ImageEditEngine = stringValueOrNull(remote.ImageEditEngine)
	state.ImageEditModel = stringValueOrNull(remote.ImageEditModel)
	state.ImageEditSize = stringValueOrNull(remote.ImageEditSize)
	state.ImagesEditOpenAIAPIBaseURL = stringValueOrNull(remote.ImagesEditOpenAIAPIBaseURL)
	state.ImagesEditOpenAIAPIKey = stringValueOrNull(remote.ImagesEditOpenAIAPIKey)
	state.ImagesEditOpenAIAPIVersion = stringValueOrNull(remote.ImagesEditOpenAIAPIVersion)
	state.ImagesEditGeminiAPIBaseURL = stringValueOrNull(remote.ImagesEditGeminiAPIBaseURL)
	state.ImagesEditGeminiAPIKey = stringValueOrNull(remote.ImagesEditGeminiAPIKey)
	state.ImagesEditComfyUIBaseURL = engineTrimmedURLValue(prior.ImagesEditComfyUIBaseURL, remote.ImagesEditComfyUIBaseURL)
	state.ImagesEditComfyUIAPIKey = stringValueOrNull(remote.ImagesEditComfyUIAPIKey)
	state.ImagesEditComfyUIWorkflow = stringValueOrNull(remote.ImagesEditComfyUIWorkflow)
	state.ImagesEditComfyUIWorkflowNodes = engineJSONValue(prior.ImagesEditComfyUIWorkflowNodes, remote.ImagesEditComfyUIWorkflowNodes, "images_edit_comfyui_workflow_nodes", diags)

	return state
}

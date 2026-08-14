resource "openwebui_tool" "example" {
  tool_id = "web_scraper"
  name    = "Web Scraper"
  content = file("${path.module}/tools/web_scraper.py")
}

# Valves are the settings the tool reads at run time. They live apart from the
# tool, so changing one does not rewrite the code.
resource "openwebui_tool_valves" "example" {
  tool_id = openwebui_tool.example.id

  valves_json = jsonencode({
    max_pages = 5
    timeout   = 30
  })
}

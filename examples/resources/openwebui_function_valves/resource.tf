resource "openwebui_function" "example" {
  function_id = "content_filter"
  name        = "Content Filter"
  content     = file("${path.module}/functions/content_filter.py")
}

# Valves are the settings the function reads at run time. They live apart from
# the function, so changing one does not rewrite the code.
resource "openwebui_function_valves" "example" {
  function_id = openwebui_function.example.id

  valves_json = jsonencode({
    priority = 5
  })
}

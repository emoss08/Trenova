data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "schema/main.go",
  ]
}
env "gorm" {
  src = data.external_schema.gorm.url
  dev = "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
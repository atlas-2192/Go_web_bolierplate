require "rake/clean"
require "fileutils"

CLOBBER.include("bin/web-starter-app", "view/.last_templ_generate", "view/*_templ.go")

file "bin/web-starter-app" => FileList["Rakefile", "*.go", "go.*", "**/*.go", "view/.last_templ_generate"].exclude(/_test.go$/) do |t|
  args = ["go", "build", "-o", t.name]

  # Uncomment the following line to enable debugging.
  # args << "-gcflags" << "all=-N -l"

  sh *args
end

# Compile all templates whenever any template changes. templ generate is fast enough that this does not slow down the
# build. If there are enough templates to cause a slowdown it may be useful to instead use a Rake rule
# (https://ruby.github.io/rake/Rake/DSL.html#method-i-rule) to only compile the templates that have changed.
file "view/.last_templ_generate" => FileList["view/*.templ"] do
  sh "templ", "generate"
  FileUtils.touch("view/.last_templ_generate")
end

desc "Build"
task build: ["bin/web-starter-app"]

desc "Run web-starter-app"
task run: :build do
  exec "bin/web-starter-app", "serve"
end

desc "Watch for source changes and rebuild and rerun"
task :rerun do
  exec "watchexec",
    "--restart",
    "--filter", "Rakefile",
    "--filter", "**/*.go",
    "--filter", "view/*.templ",
    "--ignore", "view/*_templ.go", # Ignore generated files so build doesn't trigger twice.
    "--ignore", "**/*_test.go",
    "rake", "run"
end

directory "tmp/test"

file "tmp/test/.databases-prepared" => FileList["tmp/test", "postgresql/**/*.sql", "test/testdata/*.sql"] do
  sh "psql -f test/setup_test_databases.sql > /dev/null"
  sh "touch tmp/test/.databases-prepared"
end

desc "Perform all preparation necessary to run tests"
task "test:prepare" => [:build, "tmp/test/.databases-prepared"]

desc "Run all tests"
task test: "test:prepare" do
  sh "go test ./..."
end

task default: :test

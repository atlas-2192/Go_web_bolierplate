require "rake/clean"
require "fileutils"

CLOBBER.include("bin/web-starter-app", "view/.last_templ_generate", "view/*_templ.go", "tmp/test/.databases-prepared")

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
  ENV["PGUSER"] = ENV["APP_PGUSER"] if ENV.key?("APP_PGUSER")
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

file "tmp/test/.databases-prepared" => FileList["bin/setup_test_databases", "tmp/test", "postgresql/**/*.sql", "test/testdata/*.sql"] do
  sh "bin/setup_test_databases"
  sh "touch tmp/test/.databases-prepared"
end

file "bin/setup_test_databases" => FileList["devtools/setup_test_databases/**/*.go"] do
  sh "go build -o bin/setup_test_databases github.com/jackc/web-starter-app/devtools/setup_test_databases"
end

desc "Perform all preparation necessary to run tests"
task "test:prepare" => [:build, "tmp/test/.databases-prepared"]

desc "Run all tests"
task test: "test:prepare" do
  sh "go test ./..."
end

namespace :setup do
  desc "Setup configuration files"
  task :config do
    [".envrc", "postgresql/tern.conf"].each do |filename|
      if File.exist?(filename)
        puts "Already exists: #{filename}"
      else
        FileUtils.cp("#{filename}.example", filename)
        puts "Created: #{filename}"
      end
    end

    puts
    puts "Edit these files as needed."
  end

  desc "Create a new PostgreSQL cluster for this application"
  task :create_postgresql_cluster do
    # Create the PostgreSQL cluster
    sh "initdb --locale=en_US -E UTF-8 --username=postgres .postgresql"

    # append to the configuration file
    File.open(".postgresql/postgresql.conf", "a") do |f|
      f.puts <<~TXT
        # Added by rake db:cluster:create
        #
        # Log all statements
        log_destination = 'stderr'
        log_statement = 'all'
      TXT

      if ENV.key?("PGPORT")
        f.puts <<~TXT

          port = #{ENV["PGPORT"]}
        TXT
      end
    end
  end

  desc "Setup the PostgreSQL database and user for the application"
  task :postgresql do
    sh "createdb"
    sh "createuser", ENV["APP_PGUSER"]
    sh "tern", "migrate"
  end
end

namespace :db do
  namespace :cluster do
    desc "Run the PostgreSQL cluster for this application"
    task :run do
      exec "postgres -D .postgresql"
    end
  end
end

task default: :test

use std::path::PathBuf;

use anyhow::{Context as _, Result as AnyhowResult};
use clap::{Parser, Subcommand};
#[derive(Parser, Debug)]
#[command(name = "gh-backup", about = "Backup your GitHub repositories", version)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Automatic yes to prompts; run non-interactively
    #[arg(short = 'y', long, default_value_t = false)]
    pub yes: bool,

    /// Limit of concurrent tasks
    #[arg(short = 'l', long, default_value_t = 5)]
    pub task_limit: usize,
}

impl Cli {}

/// Get the absolute path of the backup directory
pub fn get_absolute_backup_dir(backup_dir: &PathBuf) -> AnyhowResult<PathBuf> {
    let abs_path = if backup_dir.is_absolute() {
        backup_dir.clone()
    } else {
        if !backup_dir.exists() {
            std::fs::create_dir_all(backup_dir).with_context(|| {
                format!(
                    "Failed to create backup directory at {}",
                    backup_dir.display()
                )
            })?;
        }
        std::env::current_dir()
            .with_context(|| "Failed to get current working directory")?
            .join(backup_dir)
            .canonicalize()
            .with_context(|| {
                format!(
                    "Failed to canonicalize backup directory path: {}",
                    backup_dir.display()
                )
            })?
    };
    Ok(abs_path)
}

#[derive(Subcommand, Debug)]
pub enum Commands {
    Backup {
        /// GitHub username
        #[arg(short, long)]
        username: String,

        /// Backup directory
        /// Defaults to current directory/backup
        #[arg(short, long, default_value = "./backup")]
        backup_dir: PathBuf,

        /// Include all branches from repositories in the backup
        #[arg(short, long, default_value_t = true)]
        include_all_branches: bool,

        /// GitHub personal access token
        #[arg(short, long, env = "GITHUB_TOKEN", required = true)]
        token: String,

        /// Repos to exclude from backup (globs supported); specify multiple times for multiple repos
        #[arg(short, long, default_value = "", num_args = 0..)]
        exclude: Vec<String>,
    },
    /// Restore a repository working copy from a mirror backup
    Restore {
        /// Path to the mirror of the repository to restore - e.g., ./backup/repo_name.git
        #[arg(short, long)]
        path: PathBuf,

        /// Destination directory to restore the repository to
        #[arg(short, long)]
        destination: PathBuf,
    },
}

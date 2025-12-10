#![allow(unused_imports)]
use std::{
    collections::HashMap,
    io::{self, Write},
    sync::Arc,
};

use anyhow::{Context as _, Result as AnyhowResult, bail};
use clap::{Parser, Subcommand};
use glob::Pattern;

use colored::Colorize;
use futures::stream::{self, StreamExt};
use octocrab::{Page, models};

mod caching;
mod cli;
mod gh_api;
mod restore;

use crate::{
    caching::{Cache, CachedRepo},
    cli::Cli,
};

#[tokio::main]
async fn main() -> AnyhowResult<()> {
    let cli = Cli::parse();
    match &cli.command {
        cli::Commands::Backup { .. } => {}
        cli::Commands::Restore { path, destination } => {
            return restore::restore(path, destination);
        }
    }
    assert!(matches!(&cli.command, cli::Commands::Backup { .. }));

    let (cli_username, cli_backup_dir, cli_include_all_branches, cli_token, cli_exclude) =
        match &cli.command {
            cli::Commands::Backup {
                username,
                backup_dir,
                include_all_branches,
                token,
                exclude,
            } => (
                username.clone(),
                backup_dir.clone(),
                *include_all_branches,
                token.clone(),
                exclude.clone(),
            ),
            _ => unreachable!(),
        };

    let cli_exclude = cli_exclude
        .iter()
        .map(|patterns| {
            Pattern::new(patterns).with_context(|| format!("Invalid exclude pattern: {}", patterns))
        })
        .collect::<AnyhowResult<Vec<Pattern>>>()?;

    let absolute_backup_dir = cli::get_absolute_backup_dir(&cli_backup_dir)?;

    println!(
        "Backing up repositories for user: {} to {}",
        cli_username,
        absolute_backup_dir.display()
    );

    let cache =
        Arc::new(Cache::fetch(&absolute_backup_dir).with_context(|| "Failed to fetch cache")?);

    let crab = octocrab::Octocrab::builder()
        .personal_token(cli_token.clone())
        .build()
        .with_context(|| "Failed to create Octocrab instance")?;

    let repos = gh_api::get_all_repos(&crab).await?;

    let repos: Vec<models::Repository> = repos
        .into_iter()
        .filter(|repo| {
            let repo_name = repo.full_name.as_deref().unwrap_or(&repo.name);
            !cli_exclude.iter().any(|pattern| pattern.matches(repo_name))
        })
        .collect();

    if repos.is_empty() {
        println!("No repositories found to back up after applying exclude patterns.");
        return Ok(());
    }

    println!("Found {} repositories.", repos.len());
    if !cli.yes && !ask_confirmation("Do you want to proceed with the backup?") {
        println!("Backup aborted by user.");
        return Ok(());
    }

    let mut repos_indexes = HashMap::new();

    let tasks_stream = stream::iter(repos.iter().map(|repo| {
        repos_indexes.insert(repo.name.clone(), repo);
        let include_all_branches = cli_include_all_branches;
        let backup_dir = absolute_backup_dir.join(format!(
            "{}.git",
            repo.full_name
                .as_deref()
                .unwrap_or(&repo.name)
                .replace("/", "_")
        ));
        let repo_name = repo.name.clone();

        let cached_repo = cache.repos.get(&repo_name);

        async move {
            let r = backup_repo(repo, cached_repo, &backup_dir, include_all_branches).await;

            match r {
                Ok(_) => {
                    println!("Successfully backed up repository: {}", repo_name.green());
                    Ok((repo_name, r))
                }
                Err(e) => {
                    eprintln!("Failed to back up repository {}: {:?}", repo_name.red(), e);
                    Err((repo_name, e))
                }
            }
        }
    }));

    let _results: Vec<Result<_, _>> = tasks_stream
        .buffer_unordered(cli.task_limit)
        .collect()
        .await;

    /* for result in results {
        match result {
            Ok((repo_name, _)) => {
                // println!("Completed backup for repository: {}", repo_name.green());
            }
            Err((repo_name, e)) => {
                // eprintln!("Error during backup of {}: {:?}", repo_name.red(), e);
            }
        }
    } */

    Ok(())
}

async fn backup_repo(
    repo: &models::Repository,
    cached_repo: Option<&CachedRepo>,
    backup_dir: &std::path::Path,
    include_all_branches: bool,
) -> AnyhowResult<()> {
    let repo_name = repo.full_name.as_deref().unwrap_or(repo.name.as_str());
    let repo_url = match repo.clone_url.as_ref().map(|s| s.as_str()) {
        Some(url) => url,
        None => {
            bail!("Repository {} has no git URL, skipping.", repo_name.blue());
        }
    };

    if let Some(cached) = cached_repo
        && !cached.need_download(repo, backup_dir)?
    {
        println!(
            "Repository {} is up to date, skipping download.",
            repo_name.blue()
        );
        return Ok(());
    }

    // Placeholder for backup logic
    println!(
        "Backing up repo: {} to {} (include_all_branches: {})",
        repo_name.blue(),
        backup_dir.display(),
        include_all_branches
    );

    let mut cmd = tokio::process::Command::new("git");
    cmd.kill_on_drop(true);
    if !backup_dir.exists() {
        cmd.arg("clone");
        if include_all_branches {
            cmd.arg("--mirror");
        } else {
            cmd.arg("--single-branch");
        }
        cmd.arg(repo_url).arg(backup_dir);
    } else {
        cmd.current_dir(backup_dir);
        cmd.arg("fetch");
        if include_all_branches {
            cmd.arg("--all").arg("--prune");
        } else {
            cmd.arg("origin")
                .arg(repo.default_branch.as_deref().unwrap_or("main"));
        }
    }
    // println!("Executing git command for {}: {:?}", repo_name.blue(), cmd);

    let status = cmd
        .status()
        .await
        .with_context(|| format!("Failed to execute git command for {}", repo_name.blue()))?;
    if !status.success() {
        bail!(
            "Git command failed for {} with status {}",
            repo_name.blue(),
            status
        );
    }
    Ok(())
}

fn ask_confirmation(prompt: &str) -> bool {
    print!("{} [y/N]: ", prompt);
    io::stdout().flush().unwrap(); // Make sure the prompt is printed

    let mut input = String::new();
    io::stdin().read_line(&mut input).unwrap();

    matches!(input.trim().to_lowercase().as_str(), "y" | "yes")
}

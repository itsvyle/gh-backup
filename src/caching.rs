use std::{collections::HashMap, path::Path};

use anyhow::{Context as _, Result as AnyhowResult};
use octocrab::models;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct CachedRepo {
    pub full_name: String,
    // The time when this repo was last fetched
    pub last_fetched: chrono::DateTime<chrono::Utc>,
    pub hash: String,
    // These are the values that were reported by GitHub at the time of caching
    pub reported_last_push: chrono::DateTime<chrono::Utc>,
    pub reported_size: u32,
}

impl CachedRepo {
    pub fn need_download(
        &self,
        repo: &models::Repository,
        backup_dir: &Path,
    ) -> AnyhowResult<bool> {
        if !backup_dir.exists() {
            return Ok(true);
        }

        let reported_size = repo.size;
        let reported_last_push = repo.pushed_at.as_ref();
        if let Some(reported_last_push) = reported_last_push
            && let Some(reported_size) = reported_size
            && self.reported_size == reported_size
            && self.reported_last_push == *reported_last_push
        {
            Ok(false)
        } else {
            Ok(true)
        }
    }
}

pub struct Cache {
    pub repos: HashMap<String, CachedRepo>,
}

impl Cache {
    pub fn fetch(backup_dir: &Path) -> AnyhowResult<Self> {
        let cache_file = backup_dir.join("cache.json");
        if cache_file.exists() {
            let data = std::fs::read_to_string(&cache_file).with_context(|| {
                format!("Failed to read cache file at {}", cache_file.display())
            })?;
            let repos: HashMap<String, CachedRepo> =
                serde_json::from_str(&data).with_context(|| {
                    format!("Failed to parse cache file at {}", cache_file.display())
                })?;
            Ok(Cache { repos })
        } else {
            Ok(Cache {
                repos: HashMap::new(),
            })
        }
    }

    pub fn save(&self, backup_dir: &Path) -> AnyhowResult<()> {
        let cache_file = backup_dir.join("cache.json");
        let data = serde_json::to_string_pretty(&self.repos).with_context(|| {
            format!(
                "Failed to serialize cache data for saving at {}",
                cache_file.display()
            )
        })?;
        std::fs::write(&cache_file, data)
            .with_context(|| format!("Failed to write cache file at {}", cache_file.display()))?;
        Ok(())
    }
}

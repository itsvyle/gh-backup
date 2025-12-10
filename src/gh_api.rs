use anyhow::{Context as _, Result as AnyhowResult, bail};
use futures_util::TryStreamExt;
use octocrab::{Octocrab, Page, models};

pub async fn get_all_repos(crab: &Octocrab) -> AnyhowResult<Vec<models::Repository>> {
    // Placeholder for GitHub API interaction logic
    let mut repos = Vec::new();

    let page: Page<models::Repository> = crab.get("/user/repos", None::<&()>).await?;
    let stream = page.into_stream(crab);
    stream
        .try_for_each_concurrent(Some(10), |p| {
            repos.push(p);
            futures_util::future::ready(Ok(()))
        })
        .await?;

    Ok(repos)
}

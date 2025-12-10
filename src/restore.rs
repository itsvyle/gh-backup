use anyhow::{Result as AnyhowResult, bail};

pub fn restore(mirror_path: &std::path::Path, destination: &std::path::Path) -> AnyhowResult<()> {
    if !mirror_path.exists() {
        bail!("Mirror path {} does not exist.", mirror_path.display());
    }

    if destination.exists() {
        bail!("Destination path {} already exists.", destination.display());
    }

    let _status = std::process::Command::new("git")
        .arg("clone")
        .arg(mirror_path)
        .arg(destination)
        .status()
        .map_err(|e| anyhow::anyhow!("Failed to execute git command: {}", e))?;

    Ok(())
}

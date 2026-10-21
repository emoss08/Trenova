//! A service's configuration, read once at start from an optional TOML file
//! and the environment, and refused whole if any of it is wrong.
//!
//! The file is named by `--config` or by `<PREFIX>_CONFIG`. Environment
//! variables override it key by key: `<PREFIX>__POOL__WORKERS=4` sets
//! `[pool] workers = 4`. A value is read as a TOML value when it is one
//! (`4`, `true`, `"text"`, `["a", "b"]`) and as a string otherwise, so
//! `0.0.0.0:50051` needs no quotes; quote a string that would read as
//! something else (`"0x10"`).
//!
//! Decoding is strict: the configuration type denies unknown fields, so a
//! misspelt key, in the file or the environment, stops the service rather
//! than being ignored. Then [`Validate`] checks what types cannot (ranges,
//! files that must exist, settings that production requires), and every
//! problem is reported at once.

use std::fmt;
use std::path::{Path, PathBuf};

use serde::de::DeserializeOwned;
use toml::{Table, Value};

#[derive(Debug, thiserror::Error)]
pub enum ConfigError {
    #[error("read the configuration file {path}: {source}")]
    Read {
        path: PathBuf,
        source: std::io::Error,
    },
    #[error("parse the configuration file {path}: {message}")]
    Parse { path: PathBuf, message: String },
    #[error("the environment variable {name} names no configuration key")]
    EnvKey { name: String },
    #[error("the configuration is not valid: {0}")]
    Decode(String),
    #[error("the configuration is not valid:\n{0}")]
    Invalid(Validation),
}

/// Every problem validation found, one per line.
#[derive(Debug, Default)]
pub struct Validation {
    problems: Vec<String>,
}

impl Validation {
    pub fn new() -> Self {
        Self::default()
    }

    /// Records a problem with the key at `path` (`grpc.tls.cert`).
    pub fn problem(&mut self, path: &str, message: impl fmt::Display) {
        self.problems.push(format!("{path}: {message}"));
    }

    /// Records a problem unless `ok`.
    pub fn check(&mut self, ok: bool, path: &str, message: impl fmt::Display) {
        if !ok {
            self.problem(path, message);
        }
    }

    /// Records a problem unless `value` is within `range`.
    pub fn range<T: PartialOrd + Copy + fmt::Display>(
        &mut self,
        value: T,
        range: std::ops::RangeInclusive<T>,
        path: &str,
    ) {
        if !range.contains(&value) {
            let message = format!("{value} is outside {} to {}", range.start(), range.end());
            self.problem(path, message);
        }
    }

    /// Records a problem unless `path` names a readable file.
    pub fn file(&mut self, file: &Path, key: &str) {
        match std::fs::metadata(file) {
            Ok(meta) if meta.is_file() => {}
            Ok(_) => self.problem(key, format!("{} is not a file", file.display())),
            Err(err) => self.problem(key, format!("{}: {err}", file.display())),
        }
    }

    pub fn is_empty(&self) -> bool {
        self.problems.is_empty()
    }

    pub fn problems(&self) -> &[String] {
        &self.problems
    }
}

impl fmt::Display for Validation {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        for (i, problem) in self.problems.iter().enumerate() {
            if i > 0 {
                f.write_str("\n")?;
            }
            write!(f, "  - {problem}")?;
        }
        Ok(())
    }
}

/// What a configuration checks about itself after decoding.
pub trait Validate {
    fn validate(&self, v: &mut Validation);
}

/// Where a configuration is read from.
#[derive(Clone, Debug)]
pub struct Source {
    /// The environment prefix, such as `TRENOVA_IMAGING`.
    pub env_prefix: String,
    /// A file named on the command line; otherwise `<PREFIX>_CONFIG`.
    pub file: Option<PathBuf>,
}

/// Reads, decodes and validates the configuration from the process
/// environment.
pub fn load<T: DeserializeOwned + Validate>(source: &Source) -> Result<T, ConfigError> {
    load_from(source, std::env::vars())
}

/// [`load`] with the environment given, which is what tests use.
pub fn load_from<T, I>(source: &Source, env: I) -> Result<T, ConfigError>
where
    T: DeserializeOwned + Validate,
    I: IntoIterator<Item = (String, String)>,
{
    let env: Vec<(String, String)> = env.into_iter().collect();
    let file_var = format!("{}_CONFIG", source.env_prefix);
    let file = source.file.clone().or_else(|| {
        env.iter()
            .find(|(name, _)| *name == file_var)
            .map(|(_, value)| PathBuf::from(value))
    });

    let mut table = match &file {
        Some(path) => read_file(path)?,
        None => Table::new(),
    };

    let key_prefix = format!("{}__", source.env_prefix);
    for (name, raw) in &env {
        let Some(rest) = name.strip_prefix(&key_prefix) else {
            continue;
        };
        let path: Vec<String> = rest.split("__").map(str::to_ascii_lowercase).collect();
        if path.iter().any(String::is_empty) {
            return Err(ConfigError::EnvKey { name: name.clone() });
        }
        set(&mut table, &path, env_value(raw))
            .map_err(|()| ConfigError::EnvKey { name: name.clone() })?;
    }

    let config: T = Value::Table(table)
        .try_into()
        .map_err(|err: toml::de::Error| ConfigError::Decode(err.to_string()))?;

    let mut validation = Validation::new();
    config.validate(&mut validation);
    if !validation.is_empty() {
        return Err(ConfigError::Invalid(validation));
    }

    Ok(config)
}

fn read_file(path: &Path) -> Result<Table, ConfigError> {
    let text = std::fs::read_to_string(path).map_err(|source| ConfigError::Read {
        path: path.to_path_buf(),
        source,
    })?;
    text.parse::<Table>().map_err(|err| ConfigError::Parse {
        path: path.to_path_buf(),
        message: err.to_string(),
    })
}

fn env_value(raw: &str) -> Value {
    let wrapped = format!("v = {raw}");
    match wrapped.parse::<Table>() {
        Ok(mut table) => table
            .remove("v")
            .unwrap_or_else(|| Value::String(raw.to_owned())),
        Err(_) => Value::String(raw.to_owned()),
    }
}

fn set(table: &mut Table, path: &[String], value: Value) -> Result<(), ()> {
    let (last, parents) = path.split_last().ok_or(())?;
    let mut current = table;
    for key in parents {
        let entry = current
            .entry(key.clone())
            .or_insert_with(|| Value::Table(Table::new()));
        current = match entry {
            Value::Table(next) => next,
            _ => return Err(()),
        };
    }
    current.insert(last.clone(), value);
    Ok(())
}

/// Serde helpers for durations written as `"250ms"`, `"30s"`, `"5m"` or
/// `"1h"`.
pub mod duration {
    use std::time::Duration;

    use serde::{Deserialize, Deserializer, de};

    pub fn deserialize<'de, D: Deserializer<'de>>(deserializer: D) -> Result<Duration, D::Error> {
        let text = String::deserialize(deserializer)?;
        parse(&text).map_err(de::Error::custom)
    }

    /// Parses a duration with a unit: `ms`, `s`, `m` or `h`.
    pub fn parse(text: &str) -> Result<Duration, String> {
        let text = text.trim();
        let split = text
            .find(|c: char| !c.is_ascii_digit())
            .ok_or_else(|| format!("{text:?} has no unit (ms, s, m or h)"))?;
        let (number, unit) = text.split_at(split);
        let value: u64 = number
            .parse()
            .map_err(|_| format!("{text:?} is not a whole number with a unit"))?;
        match unit {
            "ms" => Ok(Duration::from_millis(value)),
            "s" => Ok(Duration::from_secs(value)),
            "m" => Ok(Duration::from_secs(value * 60)),
            "h" => Ok(Duration::from_secs(value * 3600)),
            _ => Err(format!("{text:?} has an unknown unit {unit:?}")),
        }
    }
}

/// Serde helpers for byte sizes written as `4194304`, `"64KiB"`, `"4MiB"` or
/// `"2GiB"`.
pub mod bytes {
    use serde::{Deserialize, Deserializer, de};

    #[derive(Deserialize)]
    #[serde(untagged)]
    enum Raw {
        Number(u64),
        Text(String),
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(deserializer: D) -> Result<u64, D::Error> {
        match Raw::deserialize(deserializer)? {
            Raw::Number(value) => Ok(value),
            Raw::Text(text) => parse(&text).map_err(de::Error::custom),
        }
    }

    /// Parses a size with an optional binary unit: `KiB`, `MiB` or `GiB`.
    pub fn parse(text: &str) -> Result<u64, String> {
        let text = text.trim();
        let split = text
            .find(|c: char| !c.is_ascii_digit())
            .unwrap_or(text.len());
        let (number, unit) = text.split_at(split);
        let value: u64 = number
            .parse()
            .map_err(|_| format!("{text:?} is not a size"))?;
        let factor = match unit.trim() {
            "" | "B" => 1,
            "KiB" => 1 << 10,
            "MiB" => 1 << 20,
            "GiB" => 1 << 30,
            other => return Err(format!("{text:?} has an unknown unit {other:?}")),
        };
        value
            .checked_mul(factor)
            .ok_or_else(|| format!("{text:?} is too large"))
    }
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use serde::Deserialize;

    use super::*;

    #[derive(Debug, Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Example {
        name: String,
        #[serde(default)]
        pool: Pool,
    }

    #[derive(Debug, Default, Deserialize)]
    #[serde(deny_unknown_fields)]
    struct Pool {
        #[serde(default)]
        workers: u32,
        #[serde(default, with = "duration_opt")]
        timeout: Option<Duration>,
    }

    mod duration_opt {
        use std::time::Duration;

        use serde::{Deserialize, Deserializer};

        pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Option<Duration>, D::Error> {
            let text = Option::<String>::deserialize(d)?;
            text.map(|t| super::duration::parse(&t).map_err(serde::de::Error::custom))
                .transpose()
        }
    }

    impl Validate for Example {
        fn validate(&self, v: &mut Validation) {
            v.range(self.pool.workers, 1..=64, "pool.workers");
            v.check(!self.name.is_empty(), "name", "is required");
        }
    }

    fn source() -> Source {
        Source {
            env_prefix: "TEST_SVC".into(),
            file: None,
        }
    }

    fn env(pairs: &[(&str, &str)]) -> Vec<(String, String)> {
        pairs
            .iter()
            .map(|(k, v)| ((*k).to_owned(), (*v).to_owned()))
            .collect()
    }

    #[test]
    fn the_environment_overrides_the_file_key_by_key() {
        let dir = tempfile::tempdir().expect("tempdir");
        let file = dir.path().join("svc.toml");
        std::fs::write(
            &file,
            "name = \"file\"\n[pool]\nworkers = 2\ntimeout = \"5s\"\n",
        )
        .expect("write");
        let config: Example = load_from(
            &source(),
            env(&[
                ("TEST_SVC_CONFIG", file.to_str().expect("utf-8")),
                ("TEST_SVC__POOL__WORKERS", "8"),
                ("UNRELATED", "1"),
            ]),
        )
        .expect("loads");
        assert_eq!(config.name, "file");
        assert_eq!(config.pool.workers, 8);
        assert_eq!(config.pool.timeout, Some(Duration::from_secs(5)));
    }

    #[test]
    fn a_misspelt_key_stops_the_service() {
        let err = load_from::<Example, _>(
            &source(),
            env(&[("TEST_SVC__NAME", "x"), ("TEST_SVC__POOL__WORKRS", "2")]),
        )
        .expect_err("refused");
        assert!(matches!(err, ConfigError::Decode(_)), "{err}");
    }

    #[test]
    fn every_validation_problem_is_reported_at_once() {
        let err = load_from::<Example, _>(
            &source(),
            env(&[("TEST_SVC__NAME", "\"\""), ("TEST_SVC__POOL__WORKERS", "0")]),
        )
        .expect_err("refused");
        let ConfigError::Invalid(validation) = err else {
            panic!("expected validation errors, got {err}");
        };
        assert_eq!(validation.problems().len(), 2);
    }

    #[test]
    fn unquoted_addresses_read_as_strings() {
        assert_eq!(
            env_value("0.0.0.0:50051"),
            Value::String("0.0.0.0:50051".into())
        );
        assert_eq!(env_value("4"), Value::Integer(4));
        assert_eq!(env_value("\"0x10\""), Value::String("0x10".into()));
    }

    #[test]
    fn sizes_and_durations_take_their_units() {
        assert_eq!(bytes::parse("4MiB"), Ok(4 << 20));
        assert_eq!(bytes::parse("512"), Ok(512));
        assert!(bytes::parse("4MB").is_err());
        assert_eq!(duration::parse("250ms"), Ok(Duration::from_millis(250)));
        assert_eq!(duration::parse("2m"), Ok(Duration::from_secs(120)));
        assert!(duration::parse("10").is_err());
    }
}

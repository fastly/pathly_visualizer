use std::fmt::{Display, Formatter};

#[cfg(feature = "bench")]
pub mod bench;

pub mod bzip2;

/// A quick and dirty wrapper for a value to print bytes in a more human readable form.
pub struct HumanBytes(pub u64);

impl Display for HumanBytes {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        const BYTE_SUFFIX: &[&str] = &["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
        let mut bytes_float = self.0 as f64;
        let mut suffix = 0;

        loop {
            if bytes_float < 1024.0 {
                return write!(f, "{:.3}{}", bytes_float, BYTE_SUFFIX[suffix]);
            }

            bytes_float /= 1024.0;
            suffix += 1;
        }
    }
}

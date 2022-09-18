//! Just some quick utilities I put together to assist in benchmarking various options

mod atomic;

use std::fmt::{Display, Formatter};
use std::sync::atomic::AtomicU64;
use std::sync::atomic::Ordering::SeqCst;
use std::time::{Duration, Instant};

use crate::util::bench::atomic::AtomicDuration;
#[cfg(feature = "nightly")]
pub use std::hint::black_box;

/// A stable compatible version of `std::hint::black_box`. Inspired by [rust-lang#1484] and
/// [`criterion::black_box`].
///
/// [rust-lang#1484]: https://github.com/rust-lang/rfcs/issues/1484
/// [`criterion::black_box`]: https://github.com/bheisler/criterion.rs/blob/master/src/lib.rs#L163
#[cfg(not(feature = "nightly"))]
pub fn black_box<T>(dummy: T) -> T {
    unsafe {
        let value = std::ptr::read_volatile(&dummy as *const T);
        std::mem::forget(dummy);
        value
    }
}

#[derive(Debug)]
pub struct BenchMark {
    duration: AtomicDuration,
    usages: AtomicU64,
}

impl BenchMark {
    pub fn new() -> Self {
        BenchMark {
            duration: AtomicDuration::new(Duration::from_secs(0)),
            usages: AtomicU64::new(0),
        }
    }

    #[inline]
    pub fn on<F: FnOnce() -> R, R>(&self, func: F) -> R {
        let start = Instant::now();
        let result = func();
        self.append_time(start.elapsed());
        result
    }

    pub fn append_time(&self, elapsed: Duration) {
        self.duration.fetch_add(elapsed, SeqCst);
        self.usages.fetch_add(1, SeqCst);
    }
}

impl Display for BenchMark {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let duration = self.duration.load(SeqCst);
        let usages = self.usages.load(SeqCst);

        write!(
            f,
            "{:?} over {} usages ({:?} average)",
            duration,
            usages,
            duration / usages as u32
        )
    }
}

enum ProgressWriteStrategy {
    Count(u64),
    Elapsed {
        last_print: AtomicDuration,
        period: Duration,
    },
}

pub struct ProgressCounter {
    count: AtomicU64,
    start_time: Instant,
    strategy: ProgressWriteStrategy,
}

impl Default for ProgressCounter {
    fn default() -> Self {
        ProgressCounter::every(Duration::from_secs(2))
    }
}

impl ProgressCounter {
    pub fn new(print_period: u64) -> Self {
        ProgressCounter {
            count: AtomicU64::new(0),
            start_time: Instant::now(),
            strategy: ProgressWriteStrategy::Count(print_period),
        }
    }

    pub fn every(period: Duration) -> Self {
        ProgressCounter {
            count: AtomicU64::new(0),
            start_time: Instant::now(),
            strategy: ProgressWriteStrategy::Elapsed {
                last_print: AtomicDuration::new(Duration::from_secs(0)),
                period,
            },
        }
    }

    pub fn elapsed(&self) -> Duration {
        self.start_time.elapsed()
    }

    pub fn count(&self) -> u64 {
        self.count.load(SeqCst)
    }

    pub fn periodic(&self, func: impl FnOnce(u64)) {
        let count = self.count.fetch_add(1, SeqCst) + 1;
        match &self.strategy {
            ProgressWriteStrategy::Count(period) => {
                if count % *period == 0 {
                    func(count);
                }
            }
            ProgressWriteStrategy::Elapsed { last_print, period } => {
                let prev_print_offset = last_print.load(SeqCst);

                if self.start_time + prev_print_offset + *period < Instant::now() {
                    let desired_end = Instant::now().duration_since(self.start_time);

                    // If we fail to exchange, that means another thread succeeded so there is no
                    // need to retry
                    if last_print
                        .compare_exchange(prev_print_offset, desired_end, SeqCst, SeqCst)
                        .is_ok()
                    {
                        func(count);
                    }
                }
            }
        }
    }
}

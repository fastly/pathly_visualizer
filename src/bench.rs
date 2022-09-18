//! Just some quick utilities I put together to assist in benchmarking various options

use std::fmt::{Display, Formatter};
use std::sync::atomic::AtomicU64;
use std::sync::atomic::Ordering::SeqCst;
use std::time::{Duration, Instant};

#[macro_export]
macro_rules! simple_bench {
    ($bench:ident) => {
        static $bench: $crate::bench::BenchMark = $crate::bench::BenchMark::new();
    };
    ($bench:ident: $operation:expr) => {{
        let start = ::std::time::Instant::now();
        let result = ($operation);
        $bench.append_time(start.elapsed());
        result
    }};
}

#[derive(Debug)]
pub struct BenchMark {
    duration: AtomicU64,
    usages: AtomicU64,
}

impl BenchMark {
    pub const fn new() -> Self {
        BenchMark {
            duration: AtomicU64::new(0),
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
        let nanos: u64 = elapsed.as_nanos()
            .try_into()
            .expect("Nanoseconds elapsed should fit into u64");

        self.duration.fetch_add(nanos, SeqCst);
        self.usages.fetch_add(1, SeqCst);
    }
}

impl Display for BenchMark {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let duration = Duration::from_nanos(self.duration.load(SeqCst));
        let usages = self.usages.load(SeqCst);

        write!(f, "{:?} over {} usages ({:?} average)", duration, usages, duration / usages as u32)
    }
}


enum ProgressWriteStrategy {
    Count(u64),
    Elapsed {
        last_print: AtomicU64,
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
            strategy: ProgressWriteStrategy::Elapsed{
                last_print: AtomicU64::new(0),
                period
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
                let prev_print_time = last_print.load(SeqCst);
                let last_print_offset = Duration::from_nanos(prev_print_time);

                if self.start_time + last_print_offset + *period < Instant::now() {
                    let desired_end = Instant::now().duration_since(self.start_time).as_nanos() as u64;

                    if last_print.compare_exchange(prev_print_time, desired_end, SeqCst, SeqCst).is_ok() {
                        func(count);
                    }
                }
            }
        }
    }
}

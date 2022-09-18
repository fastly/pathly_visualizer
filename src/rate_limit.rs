use std::future::Future;
use std::sync::atomic::AtomicU64;
use std::sync::atomic::Ordering::SeqCst;
use std::time::{Duration, SystemTime};

pub struct UsageLimiter<T> {
    inner: T,
    usage_limit: u64,
    limit_period: Duration,
    period_usage: AtomicU64,
    period_completed: AtomicU64,
    period_end: AtomicU64,
}

impl<T> UsageLimiter<T> {
    pub fn new(data: T, limit: u64, period: Duration) -> Self {
        // Period must be larger than the resolution of this construct
        assert!(period >= Duration::from_millis(1));

        UsageLimiter {
            inner: data,
            usage_limit: limit,
            limit_period: period,
            period_usage: AtomicU64::new(0),
            period_completed: AtomicU64::new(0),
            period_end: AtomicU64::new(0),
        }
    }

    pub async fn perform_rate_limited<F, R, A>(&self, func: F) -> R
        where F: FnOnce(&T) -> A,
              A: Future<Output=R>,
    {
        loop {
            let usage_request = self.period_usage.fetch_update(SeqCst, SeqCst, |usage| {
                (usage < self.usage_limit).then(|| usage + 1)
            });

            if usage_request.is_ok() {
                break;
            }

            if let Ok(remaining) = self.period_end().duration_since(SystemTime::now()) {
                tokio::time::sleep(remaining).await;
            }

            self.update_period_end();
        }


        let res = func(&self.inner).await;
        self.period_completed.fetch_add(1, SeqCst);
        res
    }

    #[inline]
    fn period_end(&self) -> SystemTime {
        SystemTime::UNIX_EPOCH + Duration::from_millis(self.period_end.load(SeqCst))
    }

    fn update_period_end(&self) -> SystemTime {
        let res = self.period_end.fetch_update(SeqCst, SeqCst, |prev| {
            let period_end = SystemTime::UNIX_EPOCH + Duration::from_millis(prev);

            if SystemTime::now() < period_end {
                return None;
            }

            let new_end = SystemTime::now() + self.limit_period;

            Some(new_end.duration_since(SystemTime::UNIX_EPOCH)
                .expect("Time should not be before unix epoch")
                .as_millis() as u64)
        });

        if res.is_ok() {
            let mut completed = 0;
            let _ = self.period_usage.fetch_update(SeqCst, SeqCst, |mut usage| {
                loop {
                    completed = self.period_completed.load(SeqCst);

                    // We may be overlapping with another thread in the process of doing the same
                    // thing. We need to wait until they finish updating completed to perform this
                    // operation. However we may enter a deadlock if our initial usage sample is
                    // high and has since been updated. To prevent this, re-check that we have the
                    // correct usage value on conflict.
                    if completed > usage {
                        usage = self.period_usage.load(SeqCst);
                        std::hint::spin_loop();
                        continue
                    }

                    break Some(usage - completed)
                }
            });
            self.period_completed.fetch_sub(completed, SeqCst);
        }

        let (Ok(new_end) | Err(new_end)) = res;
        SystemTime::UNIX_EPOCH + Duration::from_millis(new_end)
    }
}



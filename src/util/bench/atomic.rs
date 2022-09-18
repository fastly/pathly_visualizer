//! Okay, now hear me out. Is this over-engineered? Yes. But was it fun to write? Also yes.
//!
//! As a side note I also considered options from other crates like `atomic::Atomic<T>` and
//! `crossbeam::atomic::AtomicCell<T>`. However types like `Duration` (and `SystemTime` depending on
//! system) would still be too large for all but `AtomicU128` and require locking. Even if
//! `AtomicU128` is supported, both of these crates would still use the locking fallback instead
//! since they have alignments of less than 16 bytes.
//!
//! So what I am trying to get at is that it was completely justifiable to spend a Saturday evening
//! writing custom atomic cells for some sketchy benchmarking I occasionally use.
use std::fmt::{Debug, Display, Formatter};
use std::ops::Add;
use std::sync::atomic::*;
use std::time::Duration;

pub type AtomicDuration = AtomicCell<Duration>;

/// A `Duration` requires 96 bits to store the full precision. Ideally we would use
/// `AtomicU128`, but it is not supported on my system and I do not know how likely it is to be
/// supported on other systems.
#[cfg(target_has_atomic = "128")]
impl AtomicShim for Duration {
    type Storage = AtomicU128;

    fn to_storage(self) -> <Self::Storage as PrimitiveAtomic>::IntType {
        self.as_nanos()
    }

    fn from_storage(x: <Self::Storage as PrimitiveAtomic>::IntType) -> Self {
        let secs = u64::try_from(x / 1000_000_000).expect("Stored value too large for Duration");
        let nanos = (x % 1000_000_000) as u32;
        Duration::new(secs, nanos)
    }
}

/// Fallback to u64 since it will be supported on nearly all systems. This will limit the
/// precision of the value we store, but it should still be enough to store the nanoseconds
/// since the UNIX epoch so this is probably close enough.
#[cfg(all(target_has_atomic = "64", not(target_has_atomic = "128")))]
impl AtomicShim for Duration {
    type Storage = AtomicU64;

    fn to_storage(self) -> <Self::Storage as PrimitiveAtomic>::IntType {
        u64::try_from(self.as_nanos()).expect("Duration nanoseconds fits into u64")
    }

    fn from_storage(x: <Self::Storage as PrimitiveAtomic>::IntType) -> Self {
        Duration::from_nanos(x)
    }
}

/// I considered `crossbeam::atomic::AtomicCell`, but it would require locking. Plus it is more fun
/// to write it myself.
#[repr(transparent)]
pub struct AtomicCell<T: AtomicShim> {
    inner: <T as AtomicShim>::Storage,
}

impl<T: AtomicShim> AtomicCell<T> {
    #[inline]
    pub fn new(x: T) -> Self {
        AtomicCell {
            inner: <T as AtomicShim>::Storage::new(x.to_storage()),
        }
    }

    #[inline]
    pub fn load(&self, ordering: Ordering) -> T {
        T::from_storage(self.inner.load(ordering))
    }

    #[inline]
    pub fn store(&self, x: T, ordering: Ordering) {
        self.inner.store(x.to_storage(), ordering)
    }

    #[inline]
    pub fn swap(&self, x: T, ordering: Ordering) -> T {
        T::from_storage(self.inner.swap(x.to_storage(), ordering))
    }
}

impl<T: AtomicShim + Eq> AtomicCell<T> {
    #[inline]
    pub fn compare_exchange(
        &self,
        current: T,
        new: T,
        success: Ordering,
        failure: Ordering,
    ) -> Result<T, T> {
        self.inner
            .compare_exchange(current.to_storage(), new.to_storage(), success, failure)
            .map_err(T::from_storage)
            .map(T::from_storage)
    }

    #[inline]
    pub fn compare_exchange_weak(
        &self,
        current: T,
        new: T,
        success: Ordering,
        failure: Ordering,
    ) -> Result<T, T> {
        self.inner
            .compare_exchange_weak(current.to_storage(), new.to_storage(), success, failure)
            .map_err(T::from_storage)
            .map(T::from_storage)
    }

    #[inline]
    pub fn fetch_update<F>(
        &self,
        set_order: Ordering,
        fetch_order: Ordering,
        mut func: F,
    ) -> Result<T, T>
    where
        F: FnMut(T) -> Option<T>,
    {
        let mut previous = self.inner.load(fetch_order);
        while let Some(next) = func(T::from_storage(previous)) {
            let next_storage = next.to_storage();
            match self
                .inner
                .compare_exchange_weak(previous, next_storage, set_order, fetch_order)
            {
                // Return the previous requested instead of converting the new value to avoid errors
                // due to unexpected asymmetric conversion between storage type
                Ok(_) => return Ok(next),
                Err(e) => previous = e,
            }
        }
        Err(T::from_storage(previous))
    }
}

impl<T: AtomicShim + Add<T, Output = T>> AtomicCell<T> {
    #[inline]
    pub fn fetch_add(&self, x: T, ordering: Ordering) -> T {
        T::from_storage(self.inner.fetch_add(x.to_storage(), ordering))
    }
}

impl<T: AtomicShim + Debug> Debug for AtomicCell<T> {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "AtomicCell {{ inner: {:?} }}",
            self.load(Ordering::SeqCst)
        )
    }
}

impl<T: AtomicShim + Display> Display for AtomicCell<T> {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        self.load(Ordering::SeqCst).fmt(f)
    }
}

pub trait AtomicShim: Copy {
    type Storage: PrimitiveAtomic;

    fn to_storage(self) -> <Self::Storage as PrimitiveAtomic>::IntType;
    fn from_storage(x: <Self::Storage as PrimitiveAtomic>::IntType) -> Self;
}

pub trait PrimitiveAtomic: Default {
    type IntType: Copy;

    fn new(x: Self::IntType) -> Self;

    // Methods intrinsic to working with atomics
    fn load(&self, ordering: Ordering) -> Self::IntType;
    fn store(&self, x: Self::IntType, ordering: Ordering);
    fn compare_exchange(
        &self,
        current: Self::IntType,
        new: Self::IntType,
        success: Ordering,
        failure: Ordering,
    ) -> Result<Self::IntType, Self::IntType>;
    fn compare_exchange_weak(
        &self,
        current: Self::IntType,
        new: Self::IntType,
        success: Ordering,
        failure: Ordering,
    ) -> Result<Self::IntType, Self::IntType>;

    // Some additional methods I may or may not decide to use that might have intrinsic support
    // depending on the underlying system
    fn swap(&self, x: Self::IntType, ordering: Ordering) -> Self::IntType;
    fn fetch_add(&self, x: Self::IntType, ordering: Ordering) -> Self::IntType;
}

macro_rules! impl_prim_atomic {
    ($($cfg:meta, $prim_int:ty, $atomic:ty;)+) => {$(
        #[$cfg]
        impl PrimitiveAtomic for $atomic {
            type IntType = $prim_int;

            #[inline(always)]
            fn new(x: Self::IntType) -> Self {
                <$atomic>::new(x)
            }

            #[inline(always)]
            fn load(&self, ordering: Ordering) -> Self::IntType {
                <$atomic>::load(self, ordering)
            }

            #[inline(always)]
            fn store(&self, x: Self::IntType, ordering: Ordering) {
                <$atomic>::store(self, x, ordering)
            }

            #[inline(always)]
            fn compare_exchange(
                &self,
                current: Self::IntType,
                new: Self::IntType,
                success: Ordering,
                failure: Ordering,
            ) -> Result<Self::IntType, Self::IntType> {
                <$atomic>::compare_exchange(self, current, new, success, failure)
            }

            #[inline(always)]
            fn compare_exchange_weak(
                &self,
                current: Self::IntType,
                new: Self::IntType,
                success: Ordering,
                failure: Ordering,
            ) -> Result<Self::IntType, Self::IntType> {
                <$atomic>::compare_exchange_weak(self, current, new, success, failure)
            }

            #[inline(always)]
            fn swap(&self, x: Self::IntType, ordering: Ordering) -> Self::IntType {
                <$atomic>::swap(self, x, ordering)
            }

            #[inline(always)]
            fn fetch_add(&self, x: Self::IntType, ordering: Ordering) -> Self::IntType {
                <$atomic>::fetch_add(self, x, ordering)
            }
        }
    )+};
}

impl_prim_atomic! {
    cfg(target_has_atomic = "8"), u8, AtomicU8;
    cfg(target_has_atomic = "16"), u16, AtomicU16;
    cfg(target_has_atomic = "32"), u32, AtomicU32;
    cfg(target_has_atomic = "64"), u64, AtomicU64;
    cfg(target_has_atomic = "128"), u128, AtomicU128;
    cfg(target_has_atomic = "ptr"), usize, AtomicUsize;

    cfg(target_has_atomic = "8"), i8, AtomicI8;
    cfg(target_has_atomic = "16"), i16, AtomicI16;
    cfg(target_has_atomic = "32"), i32, AtomicI32;
    cfg(target_has_atomic = "64"), i64, AtomicI64;
    cfg(target_has_atomic = "128"), i128, AtomicI128;
    cfg(target_has_atomic = "ptr"), isize, AtomicIsize;
}

//! Assorted utility function to assist in serializing and deserializing data for some of the
//! stranger edge cases.
use serde::de::{SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};
use std::fmt;
use std::marker::PhantomData;

#[derive(Deserialize)]
#[serde(untagged)]
enum PossiblyEmpty<A> {
    Nonempty(A),
    Empty {},
}

struct ItemVisitor<'a, A> {
    _phantom: PhantomData<&'a A>,
}

impl<'de, 'a: 'de, A> Visitor<'de> for ItemVisitor<'a, A>
where
    A: Deserialize<'de>,
{
    type Value = Vec<A>;

    fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
        formatter.write_str("a list of objects")
    }

    fn visit_seq<V>(self, mut access: V) -> Result<Self::Value, V::Error>
    where
        V: SeqAccess<'de>,
    {
        let mut items = Vec::new();
        while let Some(item) = access.next_element::<PossiblyEmpty<A>>()? {
            if let PossiblyEmpty::Nonempty(value) = item {
                items.push(value);
            }
        }

        Ok(items)
    }
}

/// Deserialize a `Vec<T>` while skipping empty objects. For example `[1,2,{},3]` in JSON would be
/// treated as `[1,2,3]`.
pub fn skip_empty_in_vec<'de, D, T>(deserializer: D) -> Result<Vec<T>, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + 'de,
{
    deserializer.deserialize_seq(ItemVisitor {
        _phantom: PhantomData,
    })
}

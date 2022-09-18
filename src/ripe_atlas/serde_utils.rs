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

/// The bare minimum function to check if a value is false so that serialization can be skipped for
/// a value using `#[serde(skip_serializing_if = "is_false"]`
pub fn is_false(x: &bool) -> bool {
    !*x
}

/// Serialize/Deserialize module to have an optional comma seperated list of values.
pub mod optional_comma_seperated {
    use serde::de::{Error, Visitor};
    use serde::{Deserialize, Deserializer, Serializer};
    use std::fmt;
    use std::fmt::Debug;
    use std::marker::PhantomData;
    use std::str::FromStr;

    struct CommaSeperatedVisitor<'a, A> {
        _phantom: PhantomData<&'a A>,
    }

    impl<'de, 'a: 'de, A> Visitor<'de> for CommaSeperatedVisitor<'a, A>
    where
        A: Deserialize<'de> + FromStr,
        <A as FromStr>::Err: Debug,
    {
        type Value = Option<Vec<A>>;

        fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
            formatter.write_str("a comma seperated list of values")
        }

        fn visit_str<E>(self, v: &str) -> Result<Self::Value, E>
        where
            E: Error,
        {
            let mut items = Vec::new();

            for item in v.split(',') {
                items.push(item.parse().map_err(|e| {
                    E::custom(format!(
                        "Unable to parse item from comma seperated list: {:?}",
                        e
                    ))
                })?)
            }

            Ok(Some(items))
        }

        fn visit_none<E>(self) -> Result<Self::Value, E>
        where
            E: Error,
        {
            Ok(None)
        }
    }

    pub fn deserialize<'de, D, T>(deserializer: D) -> Result<Option<Vec<T>>, D::Error>
    where
        D: Deserializer<'de>,
        T: Deserialize<'de> + FromStr + 'de,
        <T as FromStr>::Err: Debug,
    {
        deserializer.deserialize_str(CommaSeperatedVisitor {
            _phantom: PhantomData,
        })
    }

    pub fn serialize<S, T>(this: &Option<Vec<T>>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
        T: ToString,
    {
        match this {
            Some(items) => {
                let mut buffer = String::new();

                for item in items {
                    buffer.push_str(&item.to_string());
                    buffer.push(',');
                }

                buffer.pop();
                serializer.serialize_str(&buffer)
            }
            None => serializer.serialize_none(),
        }
    }
}

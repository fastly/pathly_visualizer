use std::fmt::{Display, Formatter};
use std::num::ParseIntError;
use std::str::FromStr;

#[derive(Ord, PartialOrd, Eq, PartialEq, Hash, Debug, Copy, Clone)]
#[repr(transparent)]
pub struct IPv4Address(u32);

impl FromStr for IPv4Address {
    type Err = IPParseError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let mut iter = s.split('.');
        let mut sections = [0; 4];

        for index in 0..4 {
            sections[index] = iter
                .next()
                .ok_or(IPParseError::MissingSections)
                .and_then(|x| {
                    x.parse()
                        .map_err(|err| IPParseError::UnableToReadSection(err))
                })?;
        }

        if iter.next().is_some() {
            return Err(IPParseError::TooManySections);
        }

        Ok(IPv4Address::from(sections))
    }
}

impl Display for IPv4Address {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let [a, b, c, d] = self.0.to_be_bytes();
        write!(f, "{}.{}.{}.{}", a, b, c, d)
    }
}

impl From<[u8; 4]> for IPv4Address {
    fn from(bytes: [u8; 4]) -> Self {
        IPv4Address(u32::from_be_bytes(bytes))
    }
}

impl From<IPv4Address> for [u8; 4] {
    fn from(addr: IPv4Address) -> Self {
        addr.0.to_be_bytes()
    }
}

#[derive(Ord, PartialOrd, Eq, PartialEq, Hash, Debug, Copy, Clone)]
#[repr(transparent)]
pub struct IPv6Address([u16; 8]);

impl FromStr for IPv6Address {
    type Err = IPParseError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let mut has_omitted_zeros = false;

        if let Some(x) = s.find("::") {
            has_omitted_zeros = true;

            if Some(x) != s.rfind("::") {
                return Err(IPParseError::MultipleZerosOmitted);
            }
        }

        // A single leading or trailing colon could lead to incorrect parsing
        if (s.starts_with(':') && !s.starts_with("::")) || (s.ends_with(':') && !s.ends_with("::"))
        {
            return Err(IPParseError::MissingSections);
        }

        let mut leading_sections = 0;
        let mut trailing_sections = 0;
        let mut sections: [u16; 8] = [0; 8];

        // Parse leading sections
        for (idx, section) in s.split(':').enumerate() {
            if idx >= sections.len() {
                return Err(IPParseError::TooManySections);
            }

            if section.is_empty() {
                break;
            }

            sections[idx] = u16::from_str_radix(section, 16)
                .map_err(|err| IPParseError::UnableToReadSection(err))?;
            leading_sections += 1;
        }

        // Parse trailing sections
        for (idx, section) in s.rsplit(':').enumerate() {
            if idx >= sections.len() {
                return Err(IPParseError::TooManySections);
            }

            if section.is_empty() {
                break;
            }

            sections[sections.len() - 1 - idx] = u16::from_str_radix(section, 16)
                .map_err(|err| IPParseError::UnableToReadSection(err))?;
            trailing_sections += 1;
        }

        // Check that we have the correct number of sections
        match (has_omitted_zeros, leading_sections, trailing_sections) {
            (true, x, y) if x + y >= sections.len() => Err(IPParseError::TooManySections),
            (false, x, _) if x < sections.len() => Err(IPParseError::MissingSections),
            _ => Ok(IPv6Address::from(sections)),
        }
    }
}

impl Display for IPv6Address {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        let mut longest_zeros = 0;
        let mut longest_index = 0;

        let mut zeros_count = 0;
        for (idx, value) in self.0.into_iter().enumerate() {
            if value == 0 {
                zeros_count += 1;

                if zeros_count > longest_zeros {
                    longest_zeros = zeros_count;
                    longest_index = idx + 1 - zeros_count;
                }
            } else {
                zeros_count = 0;
            }
        }

        if longest_zeros == self.0.len() {
            return write!(f, "::");
        }

        let mut idx = 0;
        while idx < self.0.len() {
            if idx > 0 {
                write!(f, ":")?;
            }

            if idx == longest_index && longest_zeros > 0 {
                if idx == 0 || idx + longest_zeros == self.0.len() {
                    write!(f, ":")?;
                }
                idx += longest_zeros;
            } else {
                write!(f, "{:x}", self.0[idx])?;
                idx += 1;
            }
        }

        Ok(())
    }
}

impl From<[u16; 8]> for IPv6Address {
    fn from(parts: [u16; 8]) -> Self {
        IPv6Address(parts)
    }
}

impl From<IPv6Address> for [u16; 8] {
    fn from(addr: IPv6Address) -> Self {
        addr.0
    }
}

#[derive(Debug, PartialEq)]
pub enum IPParseError {
    MissingSections,
    TooManySections,
    UnableToReadSection(ParseIntError),
    /// Only possible for IPv6. Ex: "1234::abcd::4567:1"
    MultipleZerosOmitted,
}

/// Basically the same as `std::ops::RangeInclusive`, but with ordering (priority on start position).
#[derive(Ord, PartialOrd, Eq, PartialEq, Copy, Clone)]
pub struct IpRange<A> {
    pub start: A,
    pub end: A,
}

impl<A> IpRange<A> {
    pub fn new(start: A, end: A) -> Self {
        IpRange { start, end }
    }
}

impl<A: Clone> IpRange<A> {
    pub fn single(x: A) -> Self {
        IpRange {
            start: x.clone(),
            end: x,
        }
    }
}

impl<A: Ord> IpRange<A> {
    pub fn contains(&self, x: &A) -> bool {
        &self.start <= x && &self.end >= x
    }
}

// impl<A> IpRange<A> {
//     pub fn new(start: A, end: A) -> Self {
//         IpRange { start, end }
//     }
// }

// impl<A: PartialEq> PartialEq for IpRange<A> {
//     fn eq(&self, other: &Self) -> bool {
//         // self.start == other.start && self.end ==
//     }
// }
//
// impl<A: PartialEq + Eq> Eq for IpRange<A> {}
//
// impl<A: PartialEq + PartialOrd> PartialOrd for IpRange<A> {
//     fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
//         self.start.partial_cmp(&other.start)
//     }
// }
//
// impl<A: Ord + Eq + PartialEq + PartialOrd> Ord for IpRange<A> {
//     fn cmp(&self, other: &Self) -> Ordering {
//         self.start.cmp(&other.start)
//     }
// }

#[cfg(test)]
mod tests {
    use super::*;
    use std::fmt::Debug;

    /// Check that when parsed and formatted, input string remains the same
    fn check_parse_format<T>(x: &str)
    where
        T: FromStr + Debug + Display,
        <T as FromStr>::Err: Debug,
    {
        let parsed = T::from_str(x);
        assert!(parsed.is_ok(), "Failed to parse {:?}: {:?}", x, parsed);
        assert_eq!(
            format!("{}", parsed.as_ref().unwrap()),
            x,
            "Display does not match original: {:?}",
            parsed
        );
    }

    #[test]
    fn check_ipv6() {
        check_parse_format::<IPv6Address>("::");
        check_parse_format::<IPv6Address>("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff");
        check_parse_format::<IPv6Address>("::1");
        check_parse_format::<IPv6Address>("::ffff:0:0");
        check_parse_format::<IPv6Address>("64:ff9b::");
        check_parse_format::<IPv6Address>("100::ffff:ffff:ffff:ffff");
        check_parse_format::<IPv6Address>("2041:0:140f::875c:131d");
        check_parse_format::<IPv6Address>("1234:5678::9abc:0:0:def");
        check_parse_format::<IPv6Address>("1:2:3:4:5:6:7:8");

        assert_eq!(
            IPv6Address::from_str("ffff:ffff:ffff:ffff:ffff:ffff"),
            Err(IPParseError::MissingSections)
        );
        assert_eq!(
            IPv6Address::from_str("1:2:3:4:5:6:7:8:9"),
            Err(IPParseError::TooManySections)
        );
        assert!(IPv6Address::from_str("1::2:3::4").is_err());
        assert!(IPv4Address::from_str("1:2:3:4::5:6:7:8").is_err());
        assert!(IPv4Address::from_str("1").is_err());
    }

    #[test]
    fn check_ipv4() {
        check_parse_format::<IPv4Address>("0.0.0.0");
        check_parse_format::<IPv4Address>("123.234.56.78");
        check_parse_format::<IPv4Address>("255.255.255.255");

        assert_eq!(
            IPv4Address::from_str("1.2.3"),
            Err(IPParseError::MissingSections)
        );
        assert_eq!(
            IPv4Address::from_str("1.2.3.4.5"),
            Err(IPParseError::TooManySections)
        );
        assert!(IPv4Address::from_str("1.2.ff.4").is_err());
        assert!(IPv4Address::from_str("123.234.56.78-").is_err());
        assert!(IPv4Address::from_str("....").is_err());
    }
}

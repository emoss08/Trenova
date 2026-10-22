//! The front end: tokens, the syntax tree, the parser, the formatter, and loading a file with
//! the libraries it imports.

pub mod ast;
pub mod lexer;
pub mod ops;
pub mod parser;
pub mod program;
pub mod shape;
pub mod token;
pub mod walk;

pub use parser::parse;

#[cfg(test)]
mod tests;

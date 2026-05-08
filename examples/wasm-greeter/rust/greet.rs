extern crate alloc;
extern crate core;

use alloc::vec::Vec;
use std::mem::MaybeUninit;
use std::slice;

fn greet(name: &String) {
    log(&["wasm >> ", &greeting(name)].concat());
}

fn greeting(name: &String) -> String {
    ["Hello, ", &name, "!"].concat()
}

fn log(message: &String) {
    unsafe {
        let (ptr, len) = string_to_ptr(message);
        _log(ptr, len);
    }
}

#[link(wasm_import_module = "mirofish")]
extern "C" {
    #[link_name = "log"]
    fn _log(ptr: u32, size: u32);
}

#[cfg_attr(all(target_arch = "wasm32"), export_name = "mirofish_abi_version")]
pub extern "C" fn _mirofish_abi_version() -> u64 {
    1
}

#[cfg_attr(all(target_arch = "wasm32"), export_name = "greet")]
pub unsafe extern "C" fn _greet(ptr: u32, len: u32) {
    greet(&ptr_to_string(ptr, len));
}

#[cfg_attr(all(target_arch = "wasm32"), export_name = "greeting")]
pub unsafe extern "C" fn _greeting(ptr: u32, len: u32) -> u64 {
    let name = &ptr_to_string(ptr, len);
    let g = greeting(name);
    let (ptr, len) = string_to_ptr(&g);
    std::mem::forget(g);
    ((ptr as u64) << 32) | len as u64
}

unsafe fn ptr_to_string(ptr: u32, len: u32) -> String {
    let slice = slice::from_raw_parts_mut(ptr as *mut u8, len as usize);
    let utf8 = std::str::from_utf8_unchecked_mut(slice);
    String::from(utf8)
}

unsafe fn string_to_ptr(s: &String) -> (u32, u32) {
    (s.as_ptr() as u32, s.len() as u32)
}

#[cfg_attr(all(target_arch = "wasm32"), export_name = "allocate")]
pub extern "C" fn _allocate(size: u32) -> *mut u8 {
    allocate(size as usize)
}

fn allocate(size: usize) -> *mut u8 {
    let vec: Vec<MaybeUninit<u8>> = vec![MaybeUninit::uninit(); size];
    Box::into_raw(vec.into_boxed_slice()) as *mut u8
}

#[cfg_attr(all(target_arch = "wasm32"), export_name = "deallocate")]
pub unsafe extern "C" fn _deallocate(ptr: u32, size: u32) {
    deallocate(ptr as *mut u8, size as usize);
}

unsafe fn deallocate(ptr: *mut u8, size: usize) {
    let _ = Vec::from_raw_parts(ptr, 0, size);
}

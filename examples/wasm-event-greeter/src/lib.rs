use std::slice;
#[link(wasm_import_module = "mirofish")]
extern "C" {
    #[link_name = "log"]
    fn host_log(ptr: u32, size: u32);

    #[link_name = "emit_event"]
    fn host_emit_event(typ_ptr: u32, typ_len: u32, payload_ptr: u32, payload_len: u32);

    #[link_name = "time_now_unix_ms"]
    fn host_time_now_unix_ms() -> u64;
}

#[unsafe(no_mangle)]
pub extern "C" fn mirofish_abi_version() -> u64 {
    1
}

#[unsafe(no_mangle)]
pub extern "C" fn allocate(size: u32) -> *mut u8 {
    allocate_bytes(size as usize)
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn deallocate(ptr: u32, size: u32) {
    deallocate_bytes(ptr as *mut u8, size as usize)
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn run(ptr: u32, len: u32) -> u64 {
    let input = ptr_to_string(ptr, len);
    let now = host_time_now_unix_ms();

    let log_msg = format!("wasm >> handling input at {}", now);
    log_string(&log_msg);

    let event_type = "greeting.created".to_string();
    let event_payload = format!("{{\"input\":\"{}\",\"at_ms\":{}}}", input, now);
    emit_event(&event_type, &event_payload);

    let output = format!("Hello, {}!", input);
    let (out_ptr, out_len) = string_to_leaked_ptr(&output);
    ((out_ptr as u64) << 32) | out_len as u64
}

fn log_string(message: &String) {
    unsafe {
        let (ptr, len) = string_to_ptr(message);
        host_log(ptr, len);
    }
}

fn emit_event(kind: &String, payload: &String) {
    unsafe {
        let (type_ptr, type_len) = string_to_ptr(kind);
        let (payload_ptr, payload_len) = string_to_ptr(payload);
        host_emit_event(type_ptr, type_len, payload_ptr, payload_len);
    }
}

unsafe fn ptr_to_string(ptr: u32, len: u32) -> String {
    let slice = slice::from_raw_parts_mut(ptr as *mut u8, len as usize);
    let utf8 = std::str::from_utf8_unchecked_mut(slice);
    String::from(utf8)
}

unsafe fn string_to_ptr(s: &String) -> (u32, u32) {
    (s.as_ptr() as u32, s.len() as u32)
}

fn string_to_leaked_ptr(s: &String) -> (u32, u32) {
    let bytes = s.as_bytes();
    let ptr = allocate_bytes(bytes.len());
    unsafe {
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), ptr, bytes.len());
    }
    (ptr as u32, bytes.len() as u32)
}

fn allocate_bytes(size: usize) -> *mut u8 {
    let mut buf = Vec::<u8>::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

unsafe fn deallocate_bytes(ptr: *mut u8, size: usize) {
    let _ = Vec::from_raw_parts(ptr, 0, size);
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::{SystemTime, UNIX_EPOCH};

    #[test]
    fn version_is_one() {
        assert_eq!(mirofish_abi_version(), 1);
    }

    #[test]
    fn system_time_works() {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis();
        assert!(now > 0);
    }
}

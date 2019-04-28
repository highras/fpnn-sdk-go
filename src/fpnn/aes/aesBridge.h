#ifndef FPNN_GO_ENCRYPT_AES_BRIDGE_H_
#define FPNN_GO_ENCRYPT_AES_BRIDGE_H_

#include "rijndael.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef rijndael_context* ctxPtr;

ctxPtr mallocAESCtx();

#ifdef __cplusplus
}
#endif

#endif
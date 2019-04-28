#include <stdlib.h>
#include "aesBridge.h"

ctxPtr mallocAESCtx()
{
	return (rijndael_context*)malloc(sizeof(rijndael_context));
}
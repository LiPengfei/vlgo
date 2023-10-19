#ifndef _REPLACE_H_
#define _REPLACE_H_

#if defined linux

#include <dlfcn.h>
#include <stdint.h>
#include <string.h>
#include <sys/user.h>
#include <sys/mman.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

struct node {
	void *addr;	//函数入口地址
	unsigned char buf[12];//现场保存
};

struct node g_patched_list[100]; //现在最多允许patch 100个函数，足够了
int  g_patched_count =0;
void * g_handler = NULL;    // dlopen加载so的handler

// 构建Jmp指令 来达到 新函数地址 替换旧函数地址的操作
void buildJmpDirective(void *old_func, void *new_func, int save_old)
{
    // go 1.17+ use register instead of stack to pass arguments less than 9, so here only can use rdx register
    unsigned char buf[12] = {
   	                        0x48, 0xba,      // MOV rdx, buf[2:9]
   	                        0,0,0,0,0,0,0,0, // buf[2:9] will place newFunc addr
   	                        0xff, 0xe2};     // JMP [rdx]
    // copy NewFunc Addr to Buf[2:9] [one word on amd64]
    memcpy(&buf[2], &new_func, sizeof(void*));

    // add write permission to old func located page or pages
    void* page_align = (void*)((uintptr_t)old_func & PAGE_MASK);
    int pages = ((char*)old_func + 12 > (char*)page_align + PAGE_SIZE) ? 2 : 1;
    mprotect(page_align, PAGE_SIZE *pages, PROT_READ|PROT_WRITE|PROT_EXEC);

    if (save_old > 0) {
        // save old func addr to global patched list
     	struct node *pnode = &g_patched_list[g_patched_count];
     	memcpy(pnode->buf, old_func, sizeof(buf));
     	pnode->addr = old_func;

     	g_patched_count ++;
    }

    // copy new func addr to old_func position
   	memcpy(old_func, buf, sizeof(buf));
}

// 将旧函数替换成新的函数
// can refer https://bou.ke/blog/monkey-patching-in-go/
int replace( const char *funcname, void *new_func)
{
	if (new_func == NULL)
		return -1;

    // only can patch max times
	if (g_patched_count >= sizeof(g_patched_list)/sizeof(g_patched_list[0]))
		return -99;

	void *old_func = NULL;
	if (strlen(funcname)>2 && funcname[0]=='0' && funcname[1] == 'x') //支持 0x48a35d05这种绝对地址
		old_func = (void*)strtoul(funcname, NULL, 16); // read old func addr from str
	else
		old_func = dlsym(NULL, funcname);  // read old func addr by name from dynamic load symbols

	if (old_func == NULL)
		return -2;
	if (old_func == new_func)
		return -3;

    int save_old = 1;
    buildJmpDirective(old_func, new_func, save_old);

	return 0;
}

// c加载补丁
int open_patch(const char * patch_file)
{
    if(g_handler)
    {
        dlclose(g_handler);
        g_handler = NULL;
    }
    if (!patch_file || strlen(patch_file) == 0)
    {
        return -1;
    }
    g_handler = dlopen(patch_file, RTLD_NOW | RTLD_GLOBAL);
    if (g_handler == NULL)
    {
        fprintf(stderr, "dlopen failed: %s\n", dlerror());
        return -2;
    }
    // 清除可能的错误信息
    dlerror();
    return 0;
}

//找到内存映射文件加载的基地址
unsigned long FindFileMMapedBaseAddr(const char *filename)
{
    FILE * fp;
    char * line = NULL;
    size_t len = 0;
    ssize_t read;
    fp = fopen("/proc/self/maps", "r");
    if (fp == NULL)
    {
        return 0;
    }
    while((read = getline(&line, &len, fp )) != -1)
    {
        if (strstr(line, "r-xp") == NULL) continue;
        if (strstr(line, filename) == NULL) continue;
        fclose(fp);
        return strtoul(line, NULL, 16);
    }
    fclose(fp);
    return 0;
}

/**
    替换补丁中的函数地址（新版本）
    @old_func: 旧的函数地址
    @new_func: 新的函数名称
    @return: 0: 成功, 其他：失败
*/
int replace_patch_func_new(void *old_func, const char* new_func_name)
{
    if (!new_func_name || strlen(new_func_name) == 0)
        return -1;
    // 查主函数地址
    void * new_func = dlsym(NULL, new_func_name);
    if (new_func == NULL)
        return -2;
    if (g_handler == NULL || new_func == NULL || old_func == NULL)
        return -3;
    if (old_func == new_func)
        return -4;

    buildJmpDirective(old_func, new_func, 0);

	return 0;
}

/**
    替换补丁中的函数地址
    @funcname: 要替换的补丁的函数名称
    @new_func: 新的函数地址
    @return: 0: 成功, -4: patch未加载,-5:未找到补丁中的函数,-6:补丁中的函数和要替换的函数一致
*/
int replace_patch_func_withaddr(const char *funcname, void *new_func)
{
	if (g_handler == NULL || new_func == NULL)
		return -4;
	void *old_func = NULL;
	if (strlen(funcname)>2 && funcname[0]=='0' && funcname[1] == 'x') //支持 0x48a35d05这种绝对地址
	{
	    // 指定内存地址的替换
	    const char *p = strchr(funcname, '@');
	    if (p == NULL)
	    {
	        // 绝对地址，这个地址理论上应该指代主程序中的地址，后果自负
	        old_func = (void*)strtoul(funcname, NULL, 16);
	    }
	    else
	    {
	        // 指定了so的地址，需要通过/proc/self/maps去查基址
	        unsigned long base_addr = FindFileMMapedBaseAddr(p+1);
	        if (base_addr == 0)
	        {
	            // 无效的so地址
	            return -7;
	        }
	        unsigned long offset = strtoul(funcname, NULL, 16);
	        old_func = (void *)(base_addr + offset);
	    }
	}
	else
	{
	    // 指定函数名称的替换(理论上在这个patch中由于符号名一直在变动，因此基本用不到)
	    old_func = dlsym(g_handler, funcname);
	}

	if (old_func == NULL)
		return -5;
	if (old_func == new_func)
		return -6;

	buildJmpDirective(old_func, new_func, 0);

	return 0;
}

/**
    替换补丁中的函数地址
    @patch_funcname: 要替换的补丁的函数名称
    @main_funcname: 主程序中的函数名称
    @return: 0: 成功, -1:patch函数名称不合法, -2:主程序函数名称不合法, -3:查找不到主程序中的函数地址
*/
int replace_patch_func(const char *patch_funcname, const char *main_funcname)
{
    if (!patch_funcname || strlen(patch_funcname) == 0)
    {
        return -1;
    }
    if (!main_funcname || strlen(main_funcname) == 0)
    {
        return -2;
    }
    // 查主函数地址
    void * main_func_addr = dlsym(NULL, main_funcname);
    if (main_func_addr == NULL)
    {
        return -3;
    }
    return replace_patch_func_withaddr(patch_funcname, main_func_addr);
}

//还原，就像从来没发生过一样
void restore()
{
	int i;
	for(i = g_patched_count -1; i>=0; i--)
	{
		struct node *pnode = &g_patched_list[i];
		memcpy(pnode->addr, pnode->buf, sizeof(pnode->buf));
	}
	g_patched_count =0;
	if(g_handler)
	{
	    dlclose(g_handler);
	    g_handler = NULL;
	}
}
#else
//windows平台
int replace( const char *funcname, void *new_func)
{
	return -98;
}
void restore()
{
}
#endif
#endif
